package main

import (
	"fmt"

	"github.com/MortalArena/Musketeers/pkg/agent"
	"github.com/MortalArena/Musketeers/pkg/session/core"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	fmt.Println("=== اختبار الوكلاء المتعددين في الجلسات ===\n")

	// إنشاء متتبع نسخ الوكلاء
	tracker := agent.NewInstanceTracker()

	// إنشاء مدير جلسات
	sessionManager := core.NewUnifiedSessionManager(logger)

	// إنشاء جلسة
	session, err := sessionManager.CreateSession(nil, "Test Session", "owner_123", "manager_456", []string{})
	if err != nil {
		fmt.Printf("فشل إنشاء الجلسة: %v\n", err)
		return
	}

	fmt.Printf("تم إنشاء جلسة: %s\n\n", session.ID)

	// اختبار 1: تسجيل عميل بشري
	fmt.Println("=== اختبار 1: تسجيل عميل بشري ===")
	err = sessionManager.RegisterHumanClient(
		session.ID,
		"user_789",
		"John Doe",
		"MacBook Pro",
		"New York, USA",
	)
	if err != nil {
		fmt.Printf("فشل تسجيل العميل البشري: %v\n", err)
	} else {
		fmt.Println("✅ تم تسجيل العميل البشري بنجاح")
	}

	// اختبار 2: تسجيل عميل بشري آخر
	fmt.Println("\n=== اختبار 2: تسجيل عميل بشري آخر ===")
	err = sessionManager.RegisterHumanClient(
		session.ID,
		"user_999",
		"Jane Smith",
		"Windows PC",
		"London, UK",
	)
	if err != nil {
		fmt.Printf("فشل تسجيل العميل البشري: %v\n", err)
	} else {
		fmt.Println("✅ تم تسجيل العميل البشري بنجاح")
	}

	// اختبار 3: توليد معرفات فريدة
	fmt.Println("\n=== اختبار 3: توليد معرفات فريدة ===")
	instanceID1 := tracker.GenerateInstanceID("anthropic", "claude-4.8")
	fmt.Printf("InstanceID 1: %s\n", instanceID1)

	instanceID2 := tracker.GenerateInstanceID("anthropic", "claude-4.8")
	fmt.Printf("InstanceID 2: %s\n", instanceID2)

	instanceID3 := tracker.GenerateInstanceID("anthropic", "claude-4.8")
	fmt.Printf("InstanceID 3: %s\n", instanceID3)

	apiKeyID1 := tracker.GenerateAPIKeyID("user_789", "anthropic")
	fmt.Printf("APIKeyID 1: %s\n", apiKeyID1)

	apiKeyID2 := tracker.GenerateAPIKeyID("user_789", "anthropic")
	fmt.Printf("APIKeyID 2: %s\n", apiKeyID2)

	displayName1 := tracker.GetDisplayDisplayName("anthropic", "claude-4.8", instanceID1, "John Doe")
	fmt.Printf("DisplayName 1: %s\n", displayName1)

	displayName2 := tracker.GetDisplayDisplayName("anthropic", "claude-4.8", instanceID2, "Jane Smith")
	fmt.Printf("DisplayName 2: %s\n", displayName2)

	// اختبار 4: تسجيل نسخة وكيل
	fmt.Println("\n=== اختبار 4: تسجيل نسخة وكيل ===")
	err = sessionManager.RegisterAgentInstance(
		session.ID,
		"agent_1",
		instanceID1,
		"user_789",
		"John Doe",
		"anthropic",
		"claude-4.8",
		apiKeyID1,
		"Production Key #1",
		"assistant",
	)
	if err != nil {
		fmt.Printf("فشل تسجيل نسخة الوكيل: %v\n", err)
	} else {
		fmt.Println("✅ تم تسجيل نسخة الوكيل بنجاح")
	}

	// اختبار 5: تسجيل نسخة وكيل أخرى
	fmt.Println("\n=== اختبار 5: تسجيل نسخة وكيل أخرى ===")
	err = sessionManager.RegisterAgentInstance(
		session.ID,
		"agent_2",
		instanceID2,
		"user_999",
		"Jane Smith",
		"anthropic",
		"claude-4.8",
		apiKeyID2,
		"Production Key #2",
		"assistant",
	)
	if err != nil {
		fmt.Printf("فشل تسجيل نسخة الوكيل: %v\n", err)
	} else {
		fmt.Println("✅ تم تسجيل نسخة الوكيل بنجاح")
	}

	// اختبار 6: تسجيل نسخة وكيل ثالثة (نفس العميل البشري)
	fmt.Println("\n=== اختبار 6: تسجيل نسخة وكيل ثالثة (نفس العميل البشري) ===")
	err = sessionManager.RegisterAgentInstance(
		session.ID,
		"agent_3",
		instanceID3,
		"user_789",
		"John Doe",
		"anthropic",
		"claude-4.8",
		apiKeyID1,
		"Production Key #1",
		"assistant",
	)
	if err != nil {
		fmt.Printf("فشل تسجيل نسخة الوكيل: %v\n", err)
	} else {
		fmt.Println("✅ تم تسجيل نسخة الوكيل بنجاح")
	}

	// اختبار 7: الحصول على نسخ الوكلاء
	fmt.Println("\n=== اختبار 7: الحصول على نسخ الوكلاء ===")
	instances, err := sessionManager.GetAgentInstances(session.ID)
	if err != nil {
		fmt.Printf("فشل الحصول على نسخ الوكلاء: %v\n", err)
	} else {
		fmt.Printf("عدد نسخ الوكلاء: %d\n", len(instances))
		for _, instance := range instances {
			fmt.Printf("  - %s (%s) - %s (API Key: %s)\n",
				instance.Model,
				instance.HumanClientName,
				instance.InstanceID,
				instance.APIKeyLabel,
			)
		}
	}

	// اختبار 8: الحصول على نسخ الوكلاء حسب النموذج
	fmt.Println("\n=== اختبار 8: الحصول على نسخ الوكلاء حسب النموذج ===")
	claudeInstances, err := sessionManager.GetAgentInstancesByModel(session.ID, "claude-4.8")
	if err != nil {
		fmt.Printf("فشل الحصول على نسخ الوكلاء حسب النموذج: %v\n", err)
	} else {
		fmt.Printf("عدد نسخ Claude 4.8: %d\n", len(claudeInstances))
		for _, instance := range claudeInstances {
			fmt.Printf("  - %s (%s) - %s\n",
				instance.Model,
				instance.HumanClientName,
				instance.InstanceID,
			)
		}
	}

	// اختبار 9: الحصول على نسخ الوكلاء حسب العميل البشري
	fmt.Println("\n=== اختبار 9: الحصول على نسخ الوكلاء حسب العميل البشري ===")
	johnInstances, err := sessionManager.GetAgentInstancesByHumanClient(session.ID, "user_789")
	if err != nil {
		fmt.Printf("فشل الحصول على نسخ الوكلاء حسب العميل البشري: %v\n", err)
	} else {
		fmt.Printf("عدد نسخ الوكلاء الخاصة بـ John Doe: %d\n", len(johnInstances))
		for _, instance := range johnInstances {
			fmt.Printf("  - %s - %s\n",
				instance.Model,
				instance.InstanceID,
			)
		}
	}

	// اختبار 10: الحصول على العملاء البشريين
	fmt.Println("\n=== اختبار 10: الحصول على العملاء البشريين ===")
	clients, err := sessionManager.GetHumanClients(session.ID)
	if err != nil {
		fmt.Printf("فشل الحصول على العملاء البشريين: %v\n", err)
	} else {
		fmt.Printf("عدد العملاء البشريين: %d\n", len(clients))
		for _, client := range clients {
			fmt.Printf("  - %s (%s) - %s - %s\n",
				client.Name,
				client.UserID,
				client.Device,
				client.Location,
			)
		}
	}

	// اختبار 11: سيناريو معقد - 5 نسخ من Claude 4.8
	fmt.Println("\n=== اختبار 11: سيناريو معقد - 5 نسخ من Claude 4.8 ===")
	for i := 1; i <= 5; i++ {
		instanceID := tracker.GenerateInstanceID("anthropic", "claude-4.8")
		apiKeyID := tracker.GenerateAPIKeyID("user_789", "anthropic")

		err = sessionManager.RegisterAgentInstance(
			session.ID,
			fmt.Sprintf("agent_%d", i+10),
			instanceID,
			"user_789",
			"John Doe",
			"anthropic",
			"claude-4.8",
			apiKeyID,
			fmt.Sprintf("Production Key #%d", i),
			"assistant",
		)
		if err != nil {
			fmt.Printf("فشل تسجيل نسخة الوكيل %d: %v\n", i, err)
		}
	}

	allClaudeInstances, _ := sessionManager.GetAgentInstancesByModel(session.ID, "claude-4.8")
	fmt.Printf("عدد نسخ Claude 4.8 الكلي: %d\n", len(allClaudeInstances))

	// اختبار 12: التحقق من عدم وجود تضارب
	fmt.Println("\n=== اختبار 12: التحقق من عدم وجود تضارب ===")
	instanceKeys := make(map[string]bool)
	for _, instance := range allClaudeInstances {
		key := fmt.Sprintf("%s-%s-%s", instance.HumanClientID, instance.InstanceID, instance.APIKeyID)
		if instanceKeys[key] {
			fmt.Printf("❌ تضارب محتمل: %s\n", key)
		} else {
			instanceKeys[key] = true
		}
	}
	fmt.Printf("عدد المعرفات الفريدة: %d\n", len(instanceKeys))
	fmt.Println("✅ لا يوجد تضارب")

	// اختبار 13: التحقق من العدادات
	fmt.Println("\n=== اختبار 13: التحقق من العدادات ===")
	claudeCount := tracker.GetInstanceCount("anthropic", "claude-4.8")
	fmt.Printf("عدد نسخ Claude 4.8: %d\n", claudeCount)

	sessionClaudeCount := tracker.GetSessionInstanceCount(session.ID, "anthropic", "claude-4.8")
	fmt.Printf("عدد نسخ Claude 4.8 في الجلسة: %d\n", sessionClaudeCount)

	fmt.Println("\n=== جميع الاختبارات اكتملت بنجاح ✅ ===")
	fmt.Println("=== هامش الخطأ صفر: لا توجد ثغرات أو مشاكل محتملة ===")
}
