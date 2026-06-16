package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MortalArena/Musketeers/pkg/agent/integration"
	"github.com/MortalArena/Musketeers/pkg/agent_bridge"
	nrcrypto "github.com/MortalArena/Musketeers/pkg/crypto"
	"github.com/MortalArena/Musketeers/pkg/session"
	"github.com/dgraph-io/badger/v4"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

func main() {
	bridgeAddr := flag.String("bridge", "127.0.0.1:5001", "Agent Bridge address")
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
	if err := client.Connect(ctx); err != nil {
		log.Fatalf("فشل الاتصال بالجسر: %v", err)
	}
	defer client.Disconnect()

	log.WithFields(logrus.Fields{
		"did":       kp.DID,
		"bridge":    *bridgeAddr,
		"connected": client.IsConnected(),
	}).Info("Agent متصل بـ Studio Bridge")

	// [WHY] تهيئة نظام الوكيل الجماعي المتطور
	// [HOW] ينشئ نظام تكامل يربط جميع الأنظمة معاً للتعلم الجماعي
	// [SAFETY] يضمن التعلم الجماعي بين جميع الوكلاء في الجلسة

	zapLogger, _ := zap.NewProduction()
	sessionID := fmt.Sprintf("session_%d", time.Now().UnixNano())
	agentID := kp.DID

	// إنشاء قاعدة بيانات Badger للذاكرة الجماعية
	db, err := badger.Open(badger.DefaultOptions("./data/agent_memory"))
	if err != nil {
		log.Fatalf("فشل فتح قاعدة البيانات: %v", err)
	}
	defer db.Close()

	// إنشاء نظام المهارات الجماعي للجلسة
	sessionSkills := session.NewSkillsManager(sessionID)
	log.Info("تم إنشاء نظام المهارات الجماعي")

	// إنشاء الذاكرة الجماعية للجلسة
	sessionMemory := session.NewCollectiveMemory(sessionID, db)
	log.Info("تم إنشاء الذاكرة الجماعية للجلسة")

	// إنشاء نظام التكامل الجماعي
	collectiveSystem := integration.NewCollectiveAgentSystem(sessionID, sessionSkills, sessionMemory, zapLogger)
	log.Info("تم إنشاء نظام التكامل الجماعي")

	// تسجيل الوكيل في النظام الجماعي
	agentType := "coder" // يمكن تغييره حسب نوع الوكيل
	llmType := "claude"  // يمكن تغييره حسب نوع LLM
	specializations := []string{"backend", "fullstack"}
	if err := collectiveSystem.RegisterAgent(ctx, agentID, agentType, llmType, specializations); err != nil {
		log.Fatalf("فشل تسجيل الوكيل: %v", err)
	}
	log.Info("تم تسجيل الوكيل في النظام الجماعي")

	// [WHY] بدء استقبال المهام من Bridge
	// [HOW] يستقبل المهام من Bridge وينفذها باستخدام نظام التكامل الجماعي
	// [SAFETY] يستخدم goroutine لعدم حظر البرنامج الرئيسي
	go func() {
		for {
			// [HOW] في التنفيذ الحالي، الوكيل ينتظر المهام
			// في المستقبل، سيتم إضافة منطق استقبال المهام من Bridge
			// [SAFETY] يستخدم sleep لتجنب استهلاك CPU

			// محاكاة استقبال مهمة
			task := "تحليل ملفات المشروع"
			log.WithField("task", task).Info("استلمت مهمة جديدة")

			// تنفيذ المهمة باستخدام نظام التكامل الجماعي
			result, err := collectiveSystem.ExecuteTask(ctx, task, agentID)
			if err != nil {
				log.WithError(err).Error("فشل تنفيذ المهمة")
			} else {
				log.WithFields(logrus.Fields{
					"task":       task,
					"success":    result["success"],
					"duration":   result["duration"],
					"confidence": result["confidence"],
				}).Info("تم تنفيذ المهمة بنجاح")
			}

			// الحصول على ملخص النظام الجماعي
			systemSummary, _ := collectiveSystem.GetSystemSummary(ctx)
			log.WithFields(logrus.Fields{
				"system_summary": systemSummary,
			}).Info("ملخص النظام الجماعي")

			time.Sleep(5 * time.Second)
			log.Info("Agent ينتظر المهام من Bridge...")
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Info("إيقاف Agent...")
}
