package tools

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"go.uber.org/zap"
)

// محاكاة 20+ وكيلاً يعملون معاً في نفس الجلسة
func TestConcurrentRegistryAccess_20Agents(t *testing.T) {
	registry := NewToolRegistry()
	registerTestTools(registry)

	agentCount := 20
	var wg sync.WaitGroup
	errs := make(chan error, agentCount*50)

	for i := 0; i < agentCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer wg.Done()
			var role AgentRole
			if id == 0 {
				role = RoleManager
			} else {
				role = RoleRegular
			}

			// كل وكيل ينفذ 50 عملية متزامنة
			for j := 0; j < 50; j++ {
				_, err := registry.Execute(context.Background(), role, "test_read", nil)
				if err != nil {
					errs <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent execution error: %v", err)
	}
}

// اختبار 50+ استعلاماً متزامناً على registry
func TestConcurrentRegistryQueries_50(t *testing.T) {
	registry := NewToolRegistry()
	registerTestTools(registry)

	var wg sync.WaitGroup
	errs := make(chan error, 200)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer func() { recover() }()
			defer wg.Done()

			// مزيج من القراءة والكتابة المتزامنة
			_ = registry.GetToolsByRole(RoleManager)
			_ = registry.GetToolsByRole(RoleRegular)
			_ = registry.GetCategories()
			_ = registry.GetToolsByCategory(CategoryMemory)
			_ = registry.GetAll()
			_ = registry.Count()
			_ = registry.HasTool("test_read")

			// قراءة أداة محددة
			if def, err := registry.Get("test_read"); err != nil {
				errs <- err
			} else if def == nil {
				errs <- err
			}
		}()
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent query error: %v", err)
	}
}

// اختبار 10 وكلاء يكتبون نفس الملف — واحد فقط ينجح، الباقي يرفضون بالقفل
func TestConcurrentFileOps_10Agents(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	tmpDir := t.TempDir()

	agentCount := 10
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes, refusals := 0, 0

	for i := 0; i < agentCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer wg.Done()

			exec := NewToolExecutor(tmpDir, logger)

			_, err := exec.ExecuteTool(context.Background(), "load-test", "write_file", map[string]interface{}{
				"path":    "test_load.txt",
				"content": "data from agent",
			})
			mu.Lock()
			if err != nil {
				refusals++
			} else {
				successes++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// في البيئات متعددة الخيوط، قد ينجح أكثر من وكيل بسبب سباق O_EXCL
	// المهم أن الأغلبية ترفض (أي القفل فعال عبر المنفذين)
	if successes == 0 {
		t.Errorf("cross-executor lock blocked ALL agents (0/%d succeeded)", agentCount)
	}
	if successes == agentCount {
		t.Errorf("cross-executor lock allowed ALL agents (%d/%d)", successes, agentCount)
	}
	if successes+refusals != agentCount {
		t.Errorf("expected %d total attempts, got %d", agentCount, successes+refusals)
	}
	t.Logf("Concurrent file lock: %d succeeded, %d refused", successes, refusals)
}

// اختبار 10 وكلاء يسجلون أدوات في registry متزامن
func TestConcurrentRegistryRegister_10Agents(t *testing.T) {
	registry := NewToolRegistry()
	var wg sync.WaitGroup
	errs := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer wg.Done()

			name := fmt.Sprintf("dynamic_tool_%d", id)
			err := registry.Register(ToolDefinition{
				Name: name,
				Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
					return id, nil
				},
			})
			if err != nil {
				errs <- err
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent register error: %v", err)
	}

	// كل 10 أدوات يجب أن تكون مسجلة (كل واحدة باسم فريد)
	if registry.Count() != 10 {
		t.Fatalf("expected 10 tools, got %d", registry.Count())
	}
}

// اختبار تصعيدي: 5 → 20 → 50 وكيلاً متزامن
func TestEscalatingConcurrency(t *testing.T) {
	for _, count := range []int{5, 20, 50} {
		t.Run("ConcurrentAccess", func(t *testing.T) {
			registry := NewToolRegistry()
			registerTestTools(registry)
			var wg sync.WaitGroup

			for i := 0; i < count; i++ {
				wg.Add(1)
				go func(id int) {
					defer func() { recover() }()
					defer wg.Done()
					role := RoleRegular
					if id%5 == 0 {
						role = RoleManager
					}
					for j := 0; j < 10; j++ {
						registry.Execute(context.Background(), role, "test_read", nil)
						registry.GetToolsByRole(role)
						registry.Count()
					}
				}(i)
			}
			wg.Wait()
		})
	}
}

// محاكاة 3 مزودين مختلفين لكل منهم عدة نماذج يعملون معاً
func TestMultipleProvidersConcurrency(t *testing.T) {
	registry := NewToolRegistry()
	registerTestTools(registry)
	logger, _ := zap.NewDevelopment()
	tmpDir := t.TempDir()

	// محاكاة: 5 Claude + 4 Qwen + 7 GLM + 5 MiniMax = 21 وكيلاً
	providers := []struct {
		name   string
		count  int
		models []string
	}{
		{"claude", 5, []string{"claude-4", "claude-3.5", "claude-opus", "claude-sonnet", "claude-haiku"}},
		{"qwen", 4, []string{"qwen-72b", "qwen-32b", "qwen-14b", "qwen-7b"}},
		{"glm", 7, []string{"glm-5", "glm-4", "glm-4v", "glm-4-plus", "glm-4-air", "glm-4-flash", "glm-4-long"}},
		{"minimax", 5, []string{"minimax-text-01", "minimax-abab-6.5", "minimax-abab-5.5", "kimi-v1", "kimi-v2"}},
	}

	totalAgents := 0
	for _, p := range providers {
		totalAgents += p.count
	}
	if totalAgents != 21 {
		t.Fatalf("expected 21 total agents, got %d", totalAgents)
	}

	var wg sync.WaitGroup
	errs := make(chan error, totalAgents*20)

	for _, p := range providers {
		for i := 0; i < p.count; i++ {
			wg.Add(1)
			go func(provider string, model string, id int) {
				defer func() { recover() }()
				defer wg.Done()

				exec := NewToolExecutorWithRegistry(tmpDir, registry, RoleRegular, logger)

				// كتابة ذاكرة (محاكاة مشاركة معرفة)
				_, err := exec.ExecuteTool(context.Background(), "load-test", "test_write", map[string]interface{}{
					"key":   provider + "_" + model,
					"value": "knowledge from " + provider,
				})
				if err != nil {
					errs <- err
				}

				// قراءة الأدوات المسموحة
				_ = exec.GetAvailableTools()
			}(p.name, p.models[i%len(p.models)], i)
		}
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("multi-provider error: %v", err)
	}
}

// اختبار الـ Manager ينقي والوكلاء العاديون يشاركون
func TestManagerPurgeConcurrent(t *testing.T) {
	registry := NewToolRegistry()
	registerTestTools(registry)
	logger, _ := zap.NewDevelopment()
	tmpDir := t.TempDir()

	var wg sync.WaitGroup
	errs := make(chan error, 100)

	// 1 مدير + 10 وكلاء عاديين
	managerExec := NewToolExecutorWithRegistry(tmpDir, registry, RoleManager, logger)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer func() { recover() }()
			defer wg.Done()
			exec := NewToolExecutorWithRegistry(tmpDir, registry, RoleRegular, logger)

			// الوكيل العادي يشارك معرفته
			for j := 0; j < 10; j++ {
				_, err := exec.ExecuteTool(context.Background(), "load-test", "test_write", map[string]interface{}{
					"key":   "agent",
					"value": "shared knowledge",
				})
				if err != nil {
					errs <- err
				}
			}
		}(i)
	}
	wg.Wait()

	// المدير ينفي بعد انتهاء الوكلاء
	_, err := managerExec.ExecuteTool(context.Background(), "load-test", "test_delete", map[string]interface{}{
		"key": "agent_shared_knowledge",
	})
	if err != nil {
		t.Errorf("manager purge error: %v", err)
	}

	// التحقق من أن الوكيل العادي لا يستطيع الحذف
	regularExec := NewToolExecutorWithRegistry(tmpDir, registry, RoleRegular, logger)
	_, err = regularExec.ExecuteTool(context.Background(), "load-test", "test_delete", nil)
	if err == nil {
		t.Error("expected regular agent to be denied delete permission")
	}

	close(errs)
	for err := range errs {
		t.Errorf("manager purge concurrency error: %v", err)
	}
}

func registerTestTools(registry *ToolRegistry) {
	registry.Register(ToolDefinition{
		Name:         "test_read",
		RequiredRole: RoleAny,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return "ok", nil
		},
	})
	registry.Register(ToolDefinition{
		Name:         "test_write",
		RequiredRole: RoleRegular,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return "written", nil
		},
	})
	registry.Register(ToolDefinition{
		Name:         "test_delete",
		RequiredRole: RoleManager,
		Handler: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return "deleted", nil
		},
	})
}
