package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// [WHY] ToolExecutor ينفذ الأدوات مع حدود أمان
// [HOW] يفرض حدود على استدعاءات الأدوات وحجم الملفات والمسارات
// [SAFETY] يمنع الحلقات اللانهائية والوصول غير المصرح به
type ToolExecutor struct {
	// الحدود الأمان
	MaxToolCallsPerTask int           // [WHY] الحد الأقصى لاستدعاءات الأدوات (5)
	MaxFileSizeBytes    int64         // [WHY] الحد الأقصى لحجم الملف (5MB)
	AllowedBasePath     string        // [WHY] المسار المسموح (مجلد الجلسة)
	
	// حالة التنفيذ
	taskCallCount map[string]int // [WHY] عداد استدعاءات الأدوات لكل مهمة
	taskCallMu    sync.RWMutex  // [SAFETY] لحماية العدادات
	
	// Logger
	logger *zap.Logger
}

// [WHY] NewToolExecutor ينشئ منفذ أدوات جديد
// [HOW] يهيئ الحدود الأمان والعدادات
// [SAFETY] يتحقق من أن AllowedBasePath ليس فارغاً
func NewToolExecutor(allowedBasePath string, logger *zap.Logger) *ToolExecutor {
	if allowedBasePath == "" {
		allowedBasePath = "." // [SAFETY] المسار الافتراضي
	}

	return &ToolExecutor{
		MaxToolCallsPerTask: 5,            // [WHY] حد أقصى 5 استدعاءات لمنع الحلقات
		MaxFileSizeBytes:    5 * 1024 * 1024, // [WHY] 5MB كحد أقصى
		AllowedBasePath:     allowedBasePath,
		taskCallCount:       make(map[string]int),
		logger:              logger,
	}
}

// [WHY] ExecuteTool ينفذ أداة مع حدود أمان
// [HOW] يفحص العدادات والمسارات والحجم قبل التنفيذ
// [SAFETY] يمنع الحلقات اللانهائية والوصول غير المصرح به
func (te *ToolExecutor) ExecuteTool(ctx context.Context, taskID, toolName string, params map[string]interface{}) (interface{}, error) {
	// [SAFETY] فحص العدادات
	if !te.checkToolCallLimit(taskID) {
		return nil, fmt.Errorf("تجاوز الحد الأقصى لاستدعاءات الأدوات: %d", te.MaxToolCallsPerTask)
	}

	// [HOW] زيادة العداد
	te.incrementToolCallCount(taskID)

	// [SAFETY] فحص المسارات إذا كانت الأداة تتطلب مساراً
	if toolName == "read_file" || toolName == "write_file" {
		filePath, ok := params["path"].(string)
		if !ok {
			return nil, fmt.Errorf("المعامل path مطلوب")
		}

		// [SAFETY] فحص المسار
		if !te.isPathAllowed(filePath) {
			return nil, fmt.Errorf("المسار غير مسموح: %s", filePath)
		}

		// [SAFETY] فحص الحجم للقراءة
		if toolName == "read_file" {
			if err := te.checkFileSize(filePath); err != nil {
				return nil, err
			}
		}
	}

	// [HOW] تنفيذ الأداة
	result, err := te.executeToolInternal(ctx, toolName, params)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// [WHY] checkToolCallLimit يفحص ما إذا كان العداد تجاوز الحد
// [HOW] يقرأ العداد ويقارنه بالحد الأقصى
// [SAFETY] يستخدم RLock للقراءة فقط
func (te *ToolExecutor) checkToolCallLimit(taskID string) bool {
	te.taskCallMu.RLock()
	defer te.taskCallMu.RUnlock()

	count, exists := te.taskCallCount[taskID]
	if !exists {
		return true // [OK] أول استدعاء
	}

	return count < te.MaxToolCallsPerTask
}

// [WHY] incrementToolCallCount يزيد العداد
// [HOW] يقرأ العداد ويزيده
// [SAFETY] يستخدم Lock للكتابة
func (te *ToolExecutor) incrementToolCallCount(taskID string) {
	te.taskCallMu.Lock()
	defer te.taskCallMu.Unlock()

	te.taskCallCount[taskID]++
}

// [WHY] isPathAllowed يفحص ما إذا كان المسار مسموحاً
// [HOW] يفحص المسار النسبي ويمنع ../
// [SAFETY] يمنع الوصول خارج المسار المسموح
func (te *ToolExecutor) isPathAllowed(path string) bool {
	// [SAFETY] منع المسارات المطلقة
	if filepath.IsAbs(path) {
		return false
	}

	// [SAFETY] منع ../ للوصول خارج المسار المسموح
	if strings.Contains(path, "..") {
		return false
	}

	// [HOW] تحويل المسار إلى مسار مطلق
	absPath, err := filepath.Abs(filepath.Join(te.AllowedBasePath, path))
	if err != nil {
		return false
	}

	// [HOW] تحويل المسار المسموح إلى مسار مطلق
	allowedAbsPath, err := filepath.Abs(te.AllowedBasePath)
	if err != nil {
		return false
	}

	// [SAFETY] التأكد من أن المسار داخل المسار المسموح
	return strings.HasPrefix(absPath, allowedAbsPath)
}

// [WHY] checkFileSize يفحص حجم الملف
// [HOW] يقرأ معلومات الملف ويقارن الحجم
// [SAFETY] يمنع قراءة ملفات ضخمة
func (te *ToolExecutor) checkFileSize(path string) error {
	// [HOW] تحويل المسار إلى مسار مطلق
	absPath := filepath.Join(te.AllowedBasePath, path)

	// [HOW] قراءة معلومات الملف
	info, err := os.Stat(absPath)
	if err != nil {
		return err
	}

	// [SAFETY] فحص الحجم
	if info.Size() > te.MaxFileSizeBytes {
		return fmt.Errorf("حجم الملف يتجاوز الحد الأقصى: %d bytes", info.Size())
	}

	return nil
}

// [WHY] executeToolInternal ينفذ الأداة فعلياً
// [HOW] يستدعي دالة الأداة المناسبة
// [SAFETY] يستخدم context للإلغاء
func (te *ToolExecutor) executeToolInternal(ctx context.Context, toolName string, params map[string]interface{}) (interface{}, error) {
	switch toolName {
	case "read_file":
		return te.readFile(ctx, params)
	case "write_file":
		return te.writeFile(ctx, params)
	case "http_request":
		return te.httpRequest(ctx, params)
	default:
		return nil, fmt.Errorf("الأداة غير مدعومة: %s", toolName)
	}
}

// [WHY] readFile يقرأ ملف
// [HOW] يقرأ الملف ويعيد المحتوى
// [SAFETY] يستخدم context للإلغاء
func (te *ToolExecutor) readFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل path مطلوب")
	}

	// [HOW] تحويل المسار إلى مسار مطلق
	absPath := filepath.Join(te.AllowedBasePath, path)

	// [HOW] قراءة الملف مع context للإلغاء
	data, err := te.readFileWithContext(ctx, absPath)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"content": string(data),
		"path":    path,
	}, nil
}

// [WHY] readFileWithContext يقرأ ملف مع context للإلغاء
// [HOW] يقرأ الملف بشكل تدريجي مع فحص context
// [SAFETY] يلغاء القراءة إذا تم إلغاء context
func (te *ToolExecutor) readFileWithContext(ctx context.Context, path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// [HOW] قراءة بشكل تدريجي
	buf := make([]byte, 4096)
	var result []byte

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err() // [SAFETY] إلغاء القراءة
		default:
			n, err := file.Read(buf)
			if err != nil && err != io.EOF {
				return nil, err
			}
			if n == 0 {
				break
			}
			result = append(result, buf[:n]...)
		}
	}

	return result, nil
}

// [WHY] writeFile يكتب ملف
// [HOW] يكتب المحتوى إلى الملف
// [SAFETY] يستخدم context للإلغاء
func (te *ToolExecutor) writeFile(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	path, ok := params["path"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل path مطلوب")
	}

	content, ok := params["content"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل content مطلوب")
	}

	// [HOW] تحويل المسار إلى مسار مطلق
	absPath := filepath.Join(te.AllowedBasePath, path)

	// [HOW] كتابة الملف مع context للإلغاء
	err := te.writeFileWithContext(ctx, absPath, []byte(content))
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"path":    path,
	}, nil
}

// [WHY] writeFileWithContext يكتب ملف مع context للإلغاء
// [HOW] يكتب الملف بشكل تدريجي مع فحص context
// [SAFETY] يلغاء الكتابة إذا تم إلغاء context
func (te *ToolExecutor) writeFileWithContext(ctx context.Context, path string, data []byte) error {
	// [HOW] إنشاء الملف
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// [HOW] كتابة بشكل تدريجي
	chunkSize := 4096
	for i := 0; i < len(data); i += chunkSize {
		select {
		case <-ctx.Done():
			return ctx.Err() // [SAFETY] إلغاء الكتابة
		default:
			end := i + chunkSize
			if end > len(data) {
				end = len(data)
			}
			_, err := file.Write(data[i:end])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// [WHY] httpRequest يرسل طلب HTTP
// [HOW] يرسل طلب HTTP مع context للإلغاء
// [SAFETY] يلغاء الطلب إذا تم إلغاء context
func (te *ToolExecutor) httpRequest(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	url, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("المعامل url مطلوب")
	}

	method, _ := params["method"].(string)
	if method == "" {
		method = "GET"
	}

	// [HOW] إنشاء طلب مع context للإلغاء
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	// [HOW] إرسال الطلب
	client := &http.Client{
		Timeout: 30 * time.Second, // [SAFETY] مهلة 30 ثانية
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// [HOW] قراءة الاستجابة
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status_code": resp.StatusCode,
		"body":        string(body),
	}, nil
}

// [WHY] ResetTaskCallCount يصفر عداد مهمة
// [HOW] يزيل العداد من الخريطة
// [SAFETY] يستخدم Lock للكتابة
func (te *ToolExecutor) ResetTaskCallCount(taskID string) {
	te.taskCallMu.Lock()
	defer te.taskCallMu.Unlock()

	delete(te.taskCallCount, taskID)
}

// [WHY] GetTaskCallCount يحصل على عداد مهمة
// [HOW] يقرأ العداد ويعيده
// [SAFETY] يستخدم RLock للقراءة فقط
func (te *ToolExecutor) GetTaskCallCount(taskID string) int {
	te.taskCallMu.RLock()
	defer te.taskCallMu.RUnlock()

	return te.taskCallCount[taskID]
}
