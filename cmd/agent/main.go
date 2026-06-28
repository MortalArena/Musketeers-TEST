package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent/unified"
	"github.com/MortalArena/Musketeers/pkg/agent_bridge"
	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	"github.com/dgraph-io/badger/v4"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

func main() {
	bridgeAddr := flag.String("bridge", "127.0.0.1:5001", "Agent Bridge address")
	bridgeAPIKey := flag.String("bridge-api-key", "", "API key for Agent Bridge authentication")
	flag.Parse()

	log := logrus.New()
	log.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ✅ إنشاء مفاتيح للوكيل
	kp, err := nrcrypto.GenerateKeyPair()
	if err != nil {
		log.Fatalf("فشل توليد المفاتيح: %v", err)
	}

	// ✅ الاتصال بـ Agent Bridge (لا ينشئ عقدة جديدة!)
	client := agent_bridge.NewClient(*bridgeAddr, log)
	if *bridgeAPIKey != "" {
		client.SetAPIKey(*bridgeAPIKey)
		log.Info("API key set for Agent Bridge authentication")
	}
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("فشل الاتصال بالجسر: %v", err)
	}
	defer client.Disconnect()

	log.WithFields(logrus.Fields{
		"did":       kp.DID,
		"bridge":    *bridgeAddr,
		"connected": client.IsConnected(),
	}).Info("Agent متصل بـ Studio Bridge")

	// [WHY] تهيئة نظام الوكيل الموحد الذي يدمج جميع الأنظمة
	// [HOW] ينشئ نظام موحد يربط جميع الأنظمة معاً بدون تعارضات
	// [SAFETY] يضمن تناغم كامل بين جميع الأنظمة وهامش خطأ صفر

	zapLogger, _ := zap.NewProduction()
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())
	agentID := kp.DID

	// إنشاء قاعدة بيانات Badger للذاكرة الجماعية
	db, err := badger.Open(badger.DefaultOptions("./data/agent_memory"))
	if err != nil {
		log.Fatalf("فشل فتح قاعدة البيانات: %v", err)
	}
	defer db.Close()

	// إنشاء النظام الموحد الذي يدمج جميع الأنظمة
	unifiedAgent := unified.NewUnifiedAgent(sessionID, agentID, db, zapLogger)
	log.Info("تم إنشاء النظام الموحد")

	// تهيئة النظام الموحد
	if err := unifiedAgent.Initialize(ctx); err != nil {
		log.Fatalf("فشل تهيئة النظام الموحد: %v", err)
	}
	log.Info("تم تهيئة النظام الموحد بنجاح")

	// إنشاء مدير الجلسة المتطور بشكل منفصل
	sessionManager := unified.NewSessionManager(sessionID, zapLogger)
	log.Info("تم إنشاء مدير الجلسة")

	// تهيئة مدير الجلسة بتمرير UnifiedAgent كـ AgentExecutor
	if err := sessionManager.Initialize(ctx, unifiedAgent); err != nil {
		log.Fatalf("فشل تهيئة مدير الجلسة: %v", err)
	}
	log.Info("تم تهيئة مدير الجلسة بنجاح")

	// استخدام مدير الجلسة لاستقبال البرومبت
	// استقبال البرومبت من العميل
	prompt := "إنشاء نظام إدارة جلسات متطور"
	if err := sessionManager.ReceivePrompt(ctx, prompt); err != nil {
		log.WithError(err).Error("فشل استقبال البرومبت")
	} else {
		log.WithField("prompt", prompt).Info("تم استقبال البرومبت من العميل")

		// تقييم المهمة
		evaluation, err := sessionManager.EvaluateTask(ctx)
		if err != nil {
			log.WithError(err).Error("فشل تقييم المهمة")
		} else {
			log.WithFields(logrus.Fields{
				"complexity": evaluation.Complexity,
				"strategy":   evaluation.RecommendedStrategy,
			}).Info("تم تقييم المهمة")

			// تفكيك المهمة
			tasks, err := sessionManager.DecomposeTask(ctx, evaluation)
			if err != nil {
				log.WithError(err).Error("فشل تفكيك المهمة")
			} else {
				log.WithField("tasks_count", len(tasks)).Info("تم تفكيك المهمة")

				// توزيع المهام
				if err := sessionManager.DistributeTasks(ctx, tasks); err != nil {
					log.WithError(err).Error("فشل توزيع المهام")
				} else {
					log.Info("تم توزيع المهام على الوكلاء")

					// تنفيذ المهام
					if err := sessionManager.ExecuteTasks(ctx); err != nil {
						log.WithError(err).Error("فشل تنفيذ المهام")
					} else {
						log.Info("تم بدء تنفيذ المهام")
					}
				}
			}
		}
	}

	// تسجيل الوكيل في النظام الموحد
	agentType := "coder" // يمكن تغييره حسب نوع الوكيل
	llmType := "claude"  // يمكن تغييره حسب نوع LLM
	specializations := []string{"backend", "fullstack"}
	if err := unifiedAgent.RegisterAgent(ctx, agentID, agentType, llmType, specializations); err != nil {
		log.Fatalf("فشل تسجيل الوكيل: %v", err)
	}
	log.Info("تم تسجيل الوكيل في النظام الموحد")

	// [WHY] بدء استقبال المهام من Bridge
	// [HOW] يستقبل المهام من Bridge وينفذها باستخدام النظام الموحد
	// [SAFETY] يستخدم goroutine لعدم حظر البرنامج الرئيسي
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.WithField("panic", r).Error("task executor loop panicked")
			}
		}()
		for {
			// [HOW] في التنفيذ الحالي، الوكيل ينتظر المهام
			// في المستقبل، سيتم إضافة منطق استقبال المهام من Bridge
			// [SAFETY] يستخدم sleep لتجنب استهلاك CPU

			// محاكاة استقبال مهمة
			task := "تحليل ملفات المشروع"
			log.WithField("task", task).Info("استلمت مهمة جديدة")

			// تنفيذ المهمة باستخدام النظام الموحد
			result, err := unifiedAgent.ExecuteTask(ctx, task)
			if err != nil {
				log.WithError(err).Error("فشل تنفيذ المهمة")
			} else {
				log.WithFields(logrus.Fields{
					"task":       task,
					"success":    result.Success,
					"duration":   result.Duration,
					"confidence": result.Confidence,
				}).Info("تم تنفيذ المهمة بنجاح")
			}

			// الحصول على ملخص النظام الموحد
			systemSummary, _ := unifiedAgent.GetSystemSummary(ctx)
			log.WithFields(logrus.Fields{
				"system_summary": systemSummary,
			}).Info("ملخص النظام الموحد")

			time.Sleep(5 * time.Second)
			log.Info("Agent ينتظر المهام من Bridge...")
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Info("إيقاف Agent...")
}
