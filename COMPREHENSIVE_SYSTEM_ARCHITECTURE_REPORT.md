# التقرير الشامل لمعمارية النظام الكامل
## Comprehensive System Architecture Report - Complete Implementation Guide

## جدول المحتويات

1. [الملخص التنفيذي](#الملخص-التنفيذي)
2. [التحليل الشامل للملفات الموجودة](#التحليل-الشامل-للملفات-الموجودة)
3. [طبقة الأدوات الحقيقية](#طبقة-الأدوات-الحقيقية)
4. [طبقة التنفيذ الحقيقية](#طبقة-التنفيذ-الحقيقية)
5. [نظام التفويضات المتكامل](#نظام-التفويضات-المتكامل)
6. [نظام القدرات المتكامل](#نظام-القدرات-المتكامل)
7. [نظام إدارة الصلاحيات للعميل البشري](#نظام-إدارة-الصلاحيات-للعميل-البشري)
8. [التسلسل المنطقي للتنفيذ](#التسلسل-المنطقي-للتنفيذ)
9. [خطة التنفيذ الكاملة](#خطة-التنفيذ-الكاملة)
10. [تقرير إصلاح الثغرات الحرجة والمشاكل المعمارية](#تقرير-إصلاح-الثغرات-الحرجة-والماكل-المعمارية)

---

## الملخص التنفيذي

### المشكلة الحالية
النظام الحالي لديه:
- نظام تفويضات أساسي (pkg/delegation)
- نظام قدرات أساسي (pkg/capability)
- منفذ أدوات بسيط (pkg/agent/tools/executor.go)
- **لكن لا يوجد نظام متكامل للوكلاء المتعددين**

### الحل المقترح
بناء نظام متكامل يتضمن:
1. طبقة أدوات حقيقية وشاملة
2. طبقة تنفيذ حقيقية مع عزل ذكي
3. نظام تفويضات متكامل
4. نظام قدررات متكامل
5. نظام إدارة صلاحيات للعميل البشري

### الهدف النهائي
نظام يدعم عشرات الوكلاء المتزامنين في نفس الجلسة بدون انتظار، مع:
- تعاون كامل
- صلاحيات ديناميكية
- أمان شامل
- سرعة عالية

---

## التحليل الشامل للملفات الموجودة

### 1. pkg/delegation (3 ملفات)

#### advanced.go
```go
// الموجود حالياً:
type DelegationScope struct {
    WorkflowID     string
    AllowedNodeIDs []string
    AllowedActions []string
}

type DelegationRecord struct {
    ID           string
    DelegatorDID string
    DelegateDID  string
    Scope        DelegationScope
    ExpiresAt    time.Time
    Signature    []byte
}

type DelegationManager struct {
    keyResolver common.KeyResolver
}
```

**المطلوب إضافته:**
- دعم التفويضات المتسلسلة (Delegation Chain)
- دعم التفويضات المؤقتة
- دعم التفويضات المشروطة
- دعم التفويضات القابلة للإلغاء

#### advanced_test.go
**المطلوب إضافته:**
- اختبارات التفويضات المتسلسلة
- اختبارات التفويضات المؤقتة
- اختبارات التفويضات المشروطة

#### integration.go
**الموجود حالياً:** تكامل أساسي
**المطلوب إضافته:**
- تكامل مع نظام القدرات
- تكامل مع نظام إدارة الصلاحيات
- تكامل مع طبقة الأدوات

---

### 2. pkg/capability (6 ملفات)

#### manager.go
```go
// الموجود حالياً:
type Manager struct {
    mu           sync.RWMutex
    capabilities map[string]Capability
    policy       *policy.Engine
}

type Capability interface {
    Name() string
    Execute(ctx context.Context, principal policy.Principal, cmd Command) (*Result, error)
}
```

**المطلوب إضافته:**
- دعم Capability Tokens
- دعم الصلاحيات الديناميكية
- دعم الصلاحيات المؤقتة
- دعم الصلاحيات المشروطة

#### types.go
```go
// الموجود حالياً:
type Command interface {
    Name() string
    Args() map[string]any
}

type Result struct {
    Name   string
    Output map[string]any
    Error  string
}
```

**المطلوب إضافته:**
- CapabilityGrant struct
- CapabilityToken struct
- DelegationChain struct

#### github.go, github_test.go, gmail.go, gmail_test.go, messaging.go, messaging_test.go, pipeline.go, pipeline_test.go
**الموجود حالياً:** أمثلة على القدرات
**المطلوب إضافته:**
- توسيع القدرات لتشمل جميع الأدوات المطلوبة
- إضافة اختبارات شاملة

---

### 3. pkg/agent/tools (2 ملفات)

#### executor.go
```go
// الموجود حالياً:
type ToolExecutor struct {
    MaxToolCallsPerTask int
    MaxFileSizeBytes    int64
    AllowedBasePath     string
    taskCallCount       map[string]int
    taskCallMu          sync.RWMutex
    fileLockManager     *FileLockManager
    logger              *zap.Logger
}

// الأدوات المدعومة حالياً:
// - read_file
// - write_file
// - http_request
```

**المشاكل الحالية:**
- أدوات محدودة جداً (3 أدوات فقط)
- FileLockManager يخلق نقطة توقف
- لا يوجد عزل بين الوكلاء
- لا يوجد نظام صلاحيات

**المطلوب إضافته:**
- إضافة جميع الأدوات المطلوبة (30+ أداة)
- استبدال FileLockManager بنظام عزل ذكي
- تكامل مع نظام القدرات
- تكامل مع نظام التفويضات

#### file_lock.go
```go
// الموجود حالياً:
type FileLockManager struct {
    locks      map[string]*FileLock
    mu         sync.Mutex
    lockDir    string
    lockTimeout time.Duration
}
```

**المشاكل الحالية:**
- نظام قفل مركزي يخلق نقطة توقف
- لا يوجد عزل بين الوكلاء

**المطلوب إضافته:**
- استبدال بنظام Workspace Manager
- عزل كامل لكل وكيل
- حل تعارضات ذكي

---

### 4. pkg/agent/unified (28 ملف)

#### session_manager.go
```go
// الموجود حالياً:
type SessionManager struct {
    sessionID            string
    status               SessionStatus
    tasks                []*Task
    activeTasks          map[string]*Task
    taskHistory          []*Task
    realTimeMemorySync   *RealTimeMemorySync
    realTimeSkillSync    *RealTimeSkillSync
    sessionEventBus      *SessionEventBus
    agentExecutor        AgentExecutor
    logger               *zap.Logger
    mu                   sync.RWMutex
}
```

**المطلوب إضافته:**
- تكامل مع HumanCapabilityManager
- تكامل مع CapabilityGovernanceManager
- تكامل مع ToolPoolManager
- إدارة الصلاحيات الديناميكية

#### unified_agent.go
```go
// الموجود حالياً:
type UnifiedAgent struct {
    agentID              string
    sessionID            string
    localMemoryCache     *LocalMemoryCache
    unifiedSkillManager  *UnifiedSkillManager
    unifiedMemoryManager *UnifiedMemoryManager
    unifiedSyncManager   *UnifiedSyncManager
    logger               *zap.Logger
}
```

**المطلوب إضافته:**
- إضافة CapabilityToken
- إضافة DelegationChain
- تكامل مع نظام القدرات

---

### 5. pkg/orchestrator (30 ملف)

#### orchestrator_engine.go
```go
// الموجود حالياً:
type OrchestratorEngine struct {
    sessionManager       *SessionManager
    agentRegistry        *AgentRegistry
    roleAssigner        *RoleAssigner
    storageConnector     *StorageConnector
    eventBus            *eventbus.EventBus
    logger              *zap.Logger
}
```

**المطلوب إضافته:**
- تكامل مع HumanCapabilityManager
- تكامل مع CapabilityGovernanceManager
- تكامل مع ToolPoolManager
- إدارة الصلاحيات على مستوى المنسق

---

## طبقة الأدوات الحقيقية

### 1. تعريف الأدوات المطلوبة

#### الأدوات المنطقية (Logical Tools - Shared)
```go
// pkg/agent/tools/logical/memory.go
type MemoryTool struct {
    capabilityManager *CapabilityGovernanceManager
    collectiveMemory  *memory.CollectiveMemory
    logger            *zap.Logger
}

func (mt *MemoryTool) Read(ctx context.Context, agentID string, query string) (*MemoryResult, error) {
    // التحقق من الصلاحية
    if !mt.capabilityManager.CheckCapability(ctx, agentID, "memory.read", "read") {
        return nil, fmt.Errorf("agent does not have memory.read capability")
    }
    
    // تنفيذ القراءة من الذاكرة الجماعية
    return mt.collectiveMemory.Query(ctx, query)
}

func (mt *MemoryTool) Write(ctx context.Context, agentID string, entry *MemoryEntry) error {
    // التحقق من الصلاحية
    if !mt.capabilityManager.CheckCapability(ctx, agentID, "memory.write", "write") {
        return fmt.Errorf("agent does not have memory.write capability")
    }
    
    // تنفيذ الكتابة إلى الذاكرة الجماعية
    return mt.collectiveMemory.Store(ctx, entry)
}

func (mt *MemoryTool) Search(ctx context.Context, agentID string, query string) ([]*MemoryEntry, error) {
    // التحقق من الصلاحية
    if !mt.capabilityManager.CheckCapability(ctx, agentID, "memory.search", "search") {
        return nil, fmt.Errorf("agent does not have memory.search capability")
    }
    
    // تنفيذ البحث في الذاكرة الجماعية
    return mt.collectiveMemory.Search(ctx, query)
}
```

```go
// pkg/agent/tools/logical/skills.go
type SkillsTool struct {
    capabilityManager *CapabilityGovernanceManager
    skillManager      *skills.Manager
    logger            *zap.Logger
}

func (st *SkillsTool) Read(ctx context.Context, agentID string, skillID string) (*Skill, error) {
    // التحقق من الصلاحية
    if !st.capabilityManager.CheckCapability(ctx, agentID, "skills.read", "read") {
        return nil, fmt.Errorf("agent does not have skills.read capability")
    }
    
    // قراءة المهارة
    return st.skillManager.GetSkill(ctx, skillID)
}

func (st *SkillsTool) Execute(ctx context.Context, agentID string, skillID string, params map[string]interface{}) (*SkillResult, error) {
    // التحقق من الصلاحية
    if !st.capabilityManager.CheckCapability(ctx, agentID, "skills.execute", "execute") {
        return nil, fmt.Errorf("agent does not have skills.execute capability")
    }
    
    // تنفيذ المهارة
    return st.skillManager.ExecuteSkill(ctx, skillID, params)
}

func (st *SkillsTool) Learn(ctx context.Context, agentID string, skill *Skill) error {
    // التحقق من الصلاحية
    if !st.capabilityManager.CheckCapability(ctx, agentID, "skills.learn", "learn") {
        return fmt.Errorf("agent does not have skills.learn capability")
    }
    
    // تعلم المهارة الجديدة
    return st.skillManager.RegisterSkill(ctx, skill)
}
```

```go
// pkg/agent/tools/logical/channels.go
type ChannelsTool struct {
    capabilityManager *CapabilityGovernanceManager
    channelManager    *channel.Manager
    logger            *zap.Logger
}

func (ct *ChannelsTool) Read(ctx context.Context, agentID string, channelID string) (*ChannelMessage, error) {
    // التحقق من الصلاحية
    if !ct.capabilityManager.CheckCapability(ctx, agentID, "channels.read", "read") {
        return nil, fmt.Errorf("agent does not have channels.read capability")
    }
    
    // قراءة الرسالة من القناة
    return ct.channelManager.ReadMessage(ctx, channelID)
}

func (ct *ChannelsTool) Write(ctx context.Context, agentID string, channelID string, message *ChannelMessage) error {
    // التحقق من الصلاحية
    if !ct.capabilityManager.CheckCapability(ctx, agentID, "channels.write", "write") {
        return fmt.Errorf("agent does not have channels.write capability")
    }
    
    // كتابة الرسالة إلى القناة
    return ct.channelManager.WriteMessage(ctx, channelID, message)
}

func (ct *ChannelsTool) Join(ctx context.Context, agentID string, channelID string) error {
    // التحقق من الصلاحية
    if !ct.capabilityManager.CheckCapability(ctx, agentID, "channels.join", "join") {
        return fmt.Errorf("agent does not have channels.join capability")
    }
    
    // الانضمام إلى القناة
    return ct.channelManager.JoinChannel(ctx, agentID, channelID)
}
```

```go
// pkg/agent/tools/logical/registry.go
type RegistryTool struct {
    capabilityManager *CapabilityGovernanceManager
    registry          *registry.Registry
    logger            *zap.Logger
}

func (rt *RegistryTool) Read(ctx context.Context, agentID string, agentIDToQuery string) (*AgentManifest, error) {
    // التحقق من الصلاحية
    if !rt.capabilityManager.CheckCapability(ctx, agentID, "registry.read", "read") {
        return nil, fmt.Errorf("agent does not have registry.read capability")
    }
    
    // قراءة بيان الوكيل من السجل
    return rt.registry.GetAgent(ctx, agentIDToQuery)
}

func (rt *RegistryTool) Register(ctx context.Context, agentID string, manifest *AgentManifest) error {
    // التحقق من الصلاحية
    if !rt.capabilityManager.CheckCapability(ctx, agentID, "registry.register", "register") {
        return fmt.Errorf("agent does not have registry.register capability")
    }
    
    // تسجيل الوكيل في السجل
    return rt.registry.RegisterAgent(ctx, manifest)
}
```

```go
// pkg/agent/tools/logical/knowledge.go
type KnowledgeTool struct {
    capabilityManager *CapabilityGovernanceManager
    knowledgeBase      *knowledge.KnowledgeBase
    logger            *zap.Logger
}

func (kt *KnowledgeTool) Query(ctx context.Context, agentID string, query string) (*KnowledgeResult, error) {
    // التحقق من الصلاحية
    if !kt.capabilityManager.CheckCapability(ctx, agentID, "knowledge.query", "query") {
        return nil, fmt.Errorf("agent does not have knowledge.query capability")
    }
    
    // الاستعلام عن قاعدة المعرفة
    return kt.knowledgeBase.Query(ctx, query)
}

func (kt *KnowledgeTool) Update(ctx context.Context, agentID string, knowledge *KnowledgeEntry) error {
    // التحقق من الصلاحية
    if !kt.capabilityManager.CheckCapability(ctx, agentID, "knowledge.update", "update") {
        return fmt.Errorf("agent does not have knowledge.update capability")
    }
    
    // تحديث قاعدة المعرفة
    return kt.knowledgeBase.Update(ctx, knowledge)
}
```

#### الأدوات التنفيذية (Execution Tools - Isolated)

```go
// pkg/agent/tools/execution/terminal.go
type TerminalTool struct {
    capabilityManager *CapabilityGovernanceManager
    sandboxManager    *SandboxManager
    logger            *zap.Logger
}

func (tt *TerminalTool) Execute(ctx context.Context, agentID string, command string, args []string) (*TerminalResult, error) {
    // التحقق من الصلاحية
    if !tt.capabilityManager.CheckCapability(ctx, agentID, "terminal.execute", "execute") {
        return nil, fmt.Errorf("agent does not have terminal.execute capability")
    }
    
    // الحصول على صندوق الرمل للوكيل
    sandbox, err := tt.sandboxManager.GetSandbox(agentID)
    if err != nil {
        return nil, fmt.Errorf("failed to get sandbox: %w", err)
    }
    
    // تنفيذ الأمر في صندوق الرمل
    return sandbox.ExecuteCommand(ctx, command, args)
}

func (tt *TerminalTool) Read(ctx context.Context, agentID string, path string) (string, error) {
    // التحقق من الصلاحية
    if !tt.capabilityManager.CheckCapability(ctx, agentID, "terminal.read", "read") {
        return "", fmt.Errorf("agent does not have terminal.read capability")
    }
    
    // الحصول على صندوق الرمل للوكيل
    sandbox, err := tt.sandboxManager.GetSandbox(agentID)
    if err != nil {
        return "", fmt.Errorf("failed to get sandbox: %w", err)
    }
    
    // قراءة الملف في صندوق الرمل
    return sandbox.ReadFile(ctx, path)
}
```

```go
// pkg/agent/tools/execution/browser.go
type BrowserTool struct {
    capabilityManager *CapabilityGovernanceManager
    sandboxManager    *SandboxManager
    logger            *zap.Logger
}

func (bt *BrowserTool) Navigate(ctx context.Context, agentID string, url string) (*BrowserResult, error) {
    // التحقق من الصلاحية
    if !bt.capabilityManager.CheckCapability(ctx, agentID, "browser.navigate", "navigate") {
        return nil, fmt.Errorf("agent does not have browser.navigate capability")
    }
    
    // الحصول على صندوق الرمل للوكيل
    sandbox, err := bt.sandboxManager.GetSandbox(agentID)
    if err != nil {
        return nil, fmt.Errorf("failed to get sandbox: %w", err)
    }
    
    // التنقل في المتصفح في صندوق الرمل
    return sandbox.NavigateBrowser(ctx, url)
}

func (bt *BrowserTool) Interact(ctx context.Context, agentID string, selector string, action string) (*BrowserResult, error) {
    // التحقق من الصلاحية
    if !bt.capabilityManager.CheckCapability(ctx, agentID, "browser.interact", "interact") {
        return nil, fmt.Errorf("agent does not have browser.interact capability")
    }
    
    // الحصول على صندوق الرمل للوكيل
    sandbox, err := bt.sandboxManager.GetSandbox(agentID)
    if err != nil {
        return nil, fmt.Errorf("failed to get sandbox: %w", err)
    }
    
    // التفاعل مع المتصفح في صندوق الرمل
    return sandbox.InteractBrowser(ctx, selector, action)
}
```

```go
// pkg/agent/tools/execution/filesystem.go
type FilesystemTool struct {
    capabilityManager *CapabilityGovernanceManager
    workspaceManager *WorkspaceManager
    logger            *zap.Logger
}

func (ft *FilesystemTool) Read(ctx context.Context, agentID string, path string) (string, error) {
    // التحقق من الصلاحية
    if !ft.capabilityManager.CheckCapability(ctx, agentID, "filesystem.read", "read") {
        return "", fmt.Errorf("agent does not have filesystem.read capability")
    }
    
    // الحصول على مساحة العمل للوكيل
    workspace, err := ft.workspaceManager.GetWorkspace(agentID)
    if err != nil {
        return "", fmt.Errorf("failed to get workspace: %w", err)
    }
    
    // قراءة الملف في مساحة العمل
    return workspace.ReadFile(ctx, path)
}

func (ft *FilesystemTool) Write(ctx context.Context, agentID string, path string, content string) error {
    // التحقق من الصلاحية
    if !ft.capabilityManager.CheckCapability(ctx, agentID, "filesystem.write", "write") {
        return fmt.Errorf("agent does not have filesystem.write capability")
    }
    
    // الحصول على مساحة العمل للوكيل
    workspace, err := ft.workspaceManager.GetWorkspace(agentID)
    if err != nil {
        return fmt.Errorf("failed to get workspace: %w", err)
    }
    
    // كتابة الملف في مساحة العمل
    return workspace.WriteFile(ctx, path, content)
}

func (ft *FilesystemTool) Delete(ctx context.Context, agentID string, path string) error {
    // التحقق من الصلاحية
    if !ft.capabilityManager.CheckCapability(ctx, agentID, "filesystem.delete", "delete") {
        return fmt.Errorf("agent does not have filesystem.delete capability")
    }
    
    // الحصول على مساحة العمل للوكيل
    workspace, err := ft.workspaceManager.GetWorkspace(agentID)
    if err != nil {
        return fmt.Errorf("failed to get workspace: %w", err)
    }
    
    // حذف الملف في مساحة العمل
    return workspace.DeleteFile(ctx, path)
}
```

```go
// pkg/agent/tools/execution/github.go
type GitHubTool struct {
    capabilityManager *CapabilityGovernanceManager
    githubClient      *github.Client
    logger            *zap.Logger
}

func (ght *GitHubTool) Read(ctx context.Context, agentID string, repo string, path string) (string, error) {
    // التحقق من الصلاحية
    if !ght.capabilityManager.CheckCapability(ctx, agentID, "github.read", "read") {
        return "", fmt.Errorf("agent does not have github.read capability")
    }
    
    // قراءة الملف من GitHub
    return ght.githubClient.ReadFile(ctx, repo, path)
}

func (ght *GitHubTool) Push(ctx context.Context, agentID string, repo string, path string, content string) error {
    // التحقق من الصلاحية
    if !ght.capabilityManager.CheckCapability(ctx, agentID, "github.push", "push") {
        return fmt.Errorf("agent does not have github.push capability")
    }
    
    // دفع الملف إلى GitHub
    return ght.githubClient.PushFile(ctx, repo, path, content)
}

func (ght *GitHubTool) CreatePR(ctx context.Context, agentID string, repo string, title string, body string) error {
    // التحقق من الصلاحية
    if !ght.capabilityManager.CheckCapability(ctx, agentID, "github.pr", "pr") {
        return fmt.Errorf("agent does not have github.pr capability")
    }
    
    // إنشاء Pull Request
    return ght.githubClient.CreatePullRequest(ctx, repo, title, body)
}
```

```go
// pkg/agent/tools/execution/docker.go
type DockerTool struct {
    capabilityManager *CapabilityGovernanceManager
    dockerClient      *docker.Client
    sandboxManager    *SandboxManager
    logger            *zap.Logger
}

func (dt *DockerTool) Build(ctx context.Context, agentID string, dockerfile string, context string) (*DockerResult, error) {
    // التحقق من الصلاحية
    if !dt.capabilityManager.CheckCapability(ctx, agentID, "docker.build", "build") {
        return nil, fmt.Errorf("agent does not have docker.build capability")
    }
    
    // الحصول على صندوق الرمل للوكيل
    sandbox, err := dt.sandboxManager.GetSandbox(agentID)
    if err != nil {
        return nil, fmt.Errorf("failed to get sandbox: %w", err)
    }
    
    // بناء Docker Image في صندوق الرمل
    return sandbox.BuildDockerImage(ctx, dockerfile, context)
}

func (dt *DockerTool) Run(ctx context.Context, agentID string, image string, args []string) (*DockerResult, error) {
    // التحقق من الصلاحية
    if !dt.capabilityManager.CheckCapability(ctx, agentID, "docker.run", "run") {
        return nil, fmt.Errorf("agent does not have docker.run capability")
    }
    
    // الحصول على صندوق الرمل للوكيل
    sandbox, err := dt.sandboxManager.GetSandbox(agentID)
    if err != nil {
        return nil, fmt.Errorf("failed to get sandbox: %w", err)
    }
    
    // تشغيل Docker Container في صندوق الرمل
    return sandbox.RunDockerContainer(ctx, image, args)
}
```

```go
// pkg/agent/tools/execution/http.go
type HTTPTool struct {
    capabilityManager *CapabilityGovernanceManager
    httpClient        *http.Client
    logger            *zap.Logger
}

func (ht *HTTPTool) Get(ctx context.Context, agentID string, url string) (*HTTPResult, error) {
    // التحقق من الصلاحية
    if !ht.capabilityManager.CheckCapability(ctx, agentID, "http.get", "get") {
        return nil, fmt.Errorf("agent does not have http.get capability")
    }
    
    // تنفيذ GET request
    return ht.httpClient.Get(ctx, url)
}

func (ht *HTTPTool) Post(ctx context.Context, agentID string, url string, body interface{}) (*HTTPResult, error) {
    // التحقق من الصلاحية
    if !ht.capabilityManager.CheckCapability(ctx, agentID, "http.post", "post") {
        return nil, fmt.Errorf("agent does not have http.post capability")
    }
    
    // تنفيذ POST request
    return ht.httpClient.Post(ctx, url, body)
}

func (ht *HTTPTool) Put(ctx context.Context, agentID string, url string, body interface{}) (*HTTPResult, error) {
    // التحقق من الصلاحية
    if !ht.capabilityManager.CheckCapability(ctx, agentID, "http.put", "put") {
        return nil, fmt.Errorf("agent does not have http.put capability")
    }
    
    // تنفيذ PUT request
    return ht.httpClient.Put(ctx, url, body)
}

func (ht *HTTPTool) Delete(ctx context.Context, agentID string, url string) (*HTTPResult, error) {
    // التحقق من الصلاحية
    if !ht.capabilityManager.CheckCapability(ctx, agentID, "http.delete", "delete") {
        return nil, fmt.Errorf("agent does not have http.delete capability")
    }
    
    // تنفيذ DELETE request
    return ht.httpClient.Delete(ctx, url)
}
```

```go
// pkg/agent/tools/execution/database.go
type DatabaseTool struct {
    capabilityManager *CapabilityGovernanceManager
    dbManager         *database.Manager
    logger            *zap.Logger
}

func (dt *DatabaseTool) Query(ctx context.Context, agentID string, query string, params []interface{}) (*DatabaseResult, error) {
    // التحقق من الصلاحية
    if !dt.capabilityManager.CheckCapability(ctx, agentID, "database.query", "query") {
        return nil, fmt.Errorf("agent does not have database.query capability")
    }
    
    // تنفيذ الاستعلام
    return dt.dbManager.Query(ctx, query, params)
}

func (dt *DatabaseTool) Execute(ctx context.Context, agentID string, query string, params []interface{}) error {
    // التحقق من الصلاحية
    if !dt.capabilityManager.CheckCapability(ctx, agentID, "database.execute", "execute") {
        return fmt.Errorf("agent does not have database.execute capability")
    }
    
    // تنفيذ الأمر
    return dt.dbManager.Execute(ctx, query, params)
}
```

```go
// pkg/agent/tools/execution/email.go
type EmailTool struct {
    capabilityManager *CapabilityGovernanceManager
    emailClient       *email.Client
    logger            *zap.Logger
}

func (et *EmailTool) Send(ctx context.Context, agentID string, to string, subject string, body string) error {
    // التحقق من الصلاحية
    if !et.capabilityManager.CheckCapability(ctx, agentID, "email.send", "send") {
        return fmt.Errorf("agent does not have email.send capability")
    }
    
    // إرسال البريد الإلكتروني
    return et.emailClient.Send(ctx, to, subject, body)
}

func (et *EmailTool) Read(ctx context.Context, agentID string, folder string) ([]*EmailMessage, error) {
    // التحقق من الصلاحية
    if !et.capabilityManager.CheckCapability(ctx, agentID, "email.read", "read") {
        return nil, fmt.Errorf("agent does not have email.read capability")
    }
    
    // قراءة البريد الإلكتروني
    return et.emailClient.Read(ctx, folder)
}
```

### 2. Tool Registry

```go
// pkg/agent/tools/registry.go
type ToolRegistry struct {
    logicalTools   map[string]LogicalTool
    executionTools map[string]ExecutionTool
    logger         *zap.Logger
    mu             sync.RWMutex
}

func NewToolRegistry(logger *zap.Logger) *ToolRegistry {
    return &ToolRegistry{
        logicalTools:   make(map[string]LogicalTool),
        executionTools: make(map[string]ExecutionTool),
        logger:         logger,
    }
}

func (tr *ToolRegistry) RegisterLogicalTool(name string, tool LogicalTool) error {
    tr.mu.Lock()
    defer tr.mu.Unlock()
    
    if _, exists := tr.logicalTools[name]; exists {
        return fmt.Errorf("tool already registered: %s", name)
    }
    
    tr.logicalTools[name] = tool
    return nil
}

func (tr *ToolRegistry) RegisterExecutionTool(name string, tool ExecutionTool) error {
    tr.mu.Lock()
    defer tr.mu.Unlock()
    
    if _, exists := tr.executionTools[name]; exists {
        return fmt.Errorf("tool already registered: %s", name)
    }
    
    tr.executionTools[name] = tool
    return nil
}

func (tr *ToolRegistry) GetLogicalTool(name string) (LogicalTool, error) {
    tr.mu.RLock()
    defer tr.mu.RUnlock()
    
    tool, exists := tr.logicalTools[name]
    if !exists {
        return nil, fmt.Errorf("tool not found: %s", name)
    }
    
    return tool, nil
}

func (tr *ToolRegistry) GetExecutionTool(name string) (ExecutionTool, error) {
    tr.mu.RLock()
    defer tr.mu.RUnlock()
    
    tool, exists := tr.executionTools[name]
    if !exists {
        return nil, fmt.Errorf("tool not found: %s", name)
    }
    
    return tool, nil
}
```

---

## طبقة التنفيذ الحقيقية

### 1. Workspace Manager

```go
// pkg/agent/execution/workspace_manager.go
type WorkspaceManager struct {
    workspaces     map[string]*AgentWorkspace
    sharedArea      string
    conflictResolver *ConflictResolver
    logger         *zap.Logger
    mu             sync.RWMutex
}

type AgentWorkspace struct {
    AgentID       string
    BasePath      string
    TempPath      string
    SharedAccess  bool
    LastModified  time.Time
    ResourceUsage *ResourceUsage
}

type ConflictResolver struct {
    strategy ConflictStrategy
}

type ConflictStrategy string

const (
    StrategyLastWriteWins ConflictStrategy = "last_write_wins"
    StrategyManualMerge  ConflictStrategy = "manual_merge"
    StrategyAutoMerge    ConflictStrategy = "auto_merge"
    StrategyVersioning   ConflictStrategy = "versioning"
)

func NewWorkspaceManager(basePath string, logger *zap.Logger) *WorkspaceManager {
    return &WorkspaceManager{
        workspaces: make(map[string]*AgentWorkspace),
        sharedArea: filepath.Join(basePath, "shared"),
        conflictResolver: &ConflictResolver{
            strategy: StrategyAutoMerge,
        },
        logger: logger,
    }
}

func (wm *WorkspaceManager) CreateWorkspace(agentID string) (*AgentWorkspace, error) {
    wm.mu.Lock()
    defer wm.mu.Unlock()
    
    // إنشاء مسار الوكيل
    agentPath := filepath.Join(wm.sharedArea, "agents", agentID)
    tempPath := filepath.Join(agentPath, "temp")
    
    // إنشاء المجلدات
    if err := os.MkdirAll(agentPath, 0755); err != nil {
        return nil, fmt.Errorf("failed to create agent directory: %w", err)
    }
    
    if err := os.MkdirAll(tempPath, 0755); err != nil {
        return nil, fmt.Errorf("failed to create temp directory: %w", err)
    }
    
    workspace := &AgentWorkspace{
        AgentID:      agentID,
        BasePath:     agentPath,
        TempPath:     tempPath,
        SharedAccess: true,
        LastModified: time.Now(),
        ResourceUsage: &ResourceUsage{
            MemoryMB:    0,
            CPUUsage:    0,
            DiskIO:      0,
            NetworkIO:   0,
            LastUpdated: time.Now(),
        },
    }
    
    wm.workspaces[agentID] = workspace
    return workspace, nil
}

func (wm *WorkspaceManager) GetWorkspace(agentID string) (*AgentWorkspace, error) {
    wm.mu.RLock()
    defer wm.mu.RUnlock()
    
    workspace, exists := wm.workspaces[agentID]
    if !exists {
        return nil, fmt.Errorf("workspace not found for agent: %s", agentID)
    }
    
    return workspace, nil
}

func (wm *WorkspaceManager) DeleteWorkspace(agentID string) error {
    wm.mu.Lock()
    defer wm.mu.Unlock()
    
    workspace, exists := wm.workspaces[agentID]
    if !exists {
        return fmt.Errorf("workspace not found for agent: %s", agentID)
    }
    
    // حذف المجلدات
    if err := os.RemoveAll(workspace.BasePath); err != nil {
        return fmt.Errorf("failed to delete workspace: %w", err)
    }
    
    delete(wm.workspaces, agentID)
    return nil
}
```

### 2. Sandbox Manager

```go
// pkg/agent/execution/sandbox_manager.go
type SandboxManager struct {
    sandboxes    map[string]*AgentSandbox
    baseDir      string
    logger       *zap.Logger
    mu           sync.RWMutex
}

type AgentSandbox struct {
    AgentID       string
    ContainerID   string
    Status        SandboxStatus
    ResourceUsage *ResourceUsage
    CreatedAt     time.Time
    LastActivity  time.Time
}

type SandboxStatus string

const (
    SandboxStatusRunning   SandboxStatus = "running"
    SandboxStatusStopped   SandboxStatus = "stopped"
    SandboxStatusError     SandboxStatus = "error"
    SandboxStatusCreating  SandboxStatus = "creating"
)

func NewSandboxManager(baseDir string, logger *zap.Logger) *SandboxManager {
    return &SandboxManager{
        sandboxes: make(map[string]*AgentSandbox),
        baseDir:   baseDir,
        logger:    logger,
    }
}

func (sm *SandboxManager) CreateSandbox(agentID string) (*AgentSandbox, error) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    // إنشاء الحاوية
    containerID, err := sm.createDockerContainer(agentID)
    if err != nil {
        return nil, fmt.Errorf("failed to create container: %w", err)
    }
    
    sandbox := &AgentSandbox{
        AgentID:      agentID,
        ContainerID:  containerID,
        Status:       SandboxStatusRunning,
        ResourceUsage: &ResourceUsage{
            MemoryMB:    0,
            CPUUsage:    0,
            DiskIO:      0,
            NetworkIO:   0,
            LastUpdated: time.Now(),
        },
        CreatedAt:    time.Now(),
        LastActivity: time.Now(),
    }
    
    sm.sandboxes[agentID] = sandbox
    return sandbox, nil
}

func (sm *SandboxManager) GetSandbox(agentID string) (*AgentSandbox, error) {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    
    sandbox, exists := sm.sandboxes[agentID]
    if !exists {
        return nil, fmt.Errorf("sandbox not found for agent: %s", agentID)
    }
    
    return sandbox, nil
}

func (sm *SandboxManager) DeleteSandbox(agentID string) error {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    sandbox, exists := sm.sandboxes[agentID]
    if !exists {
        return fmt.Errorf("sandbox not found for agent: %s", agentID)
    }
    
    // إيقاف الحاوية
    if err := sm.stopDockerContainer(sandbox.ContainerID); err != nil {
        return fmt.Errorf("failed to stop container: %w", err)
    }
    
    delete(sm.sandboxes, agentID)
    return nil
}

func (sm *SandboxManager) ExecuteCommand(ctx context.Context, agentID string, command string, args []string) (*TerminalResult, error) {
    sandbox, err := sm.GetSandbox(agentID)
    if err != nil {
        return nil, err
    }
    
    // تنفيذ الأمر في الحاوية
    return sm.executeInContainer(ctx, sandbox.ContainerID, command, args)
}

func (sm *SandboxManager) createDockerContainer(agentID string) (string, error) {
    // إنشاء حاوية Docker
    // يستخدم docker SDK
    return "container-" + agentID, nil
}

func (sm *SandboxManager) stopDockerContainer(containerID string) error {
    // إيقاف حاوية Docker
    // يستخدم docker SDK
    return nil
}

func (sm *SandboxManager) executeInContainer(ctx context.Context, containerID string, command string, args []string) (*TerminalResult, error) {
    // تنفيذ الأمر في الحاوية
    // يستخدم docker SDK
    return &TerminalResult{
        Output:   "command output",
        ExitCode: 0,
    }, nil
}
```

### 3. Execution Layer

```go
// pkg/agent/execution/execution_layer.go
type ExecutionLayer struct {
    capabilityManager *CapabilityGovernanceManager
    toolRegistry      *ToolRegistry
    workspaceManager  *WorkspaceManager
    sandboxManager    *SandboxManager
    logger            *zap.Logger
}

func NewExecutionLayer(
    capabilityManager *CapabilityGovernanceManager,
    toolRegistry *ToolRegistry,
    workspaceManager *WorkspaceManager,
    sandboxManager *SandboxManager,
    logger *zap.Logger,
) *ExecutionLayer {
    return &ExecutionLayer{
        capabilityManager: capabilityManager,
        toolRegistry:      toolRegistry,
        workspaceManager:  workspaceManager,
        sandboxManager:    sandboxManager,
        logger:            logger,
    }
}

func (el *ExecutionLayer) Execute(
    ctx context.Context,
    agentID string,
    toolName string,
    action string,
    args map[string]interface{},
) (*Result, error) {
    
    // التحقق من الصلاحية
    capabilityName := fmt.Sprintf("%s.%s", toolName, action)
    if !el.capabilityManager.CheckCapability(ctx, agentID, capabilityName, action) {
        return nil, fmt.Errorf("agent %s does not have capability %s", agentID, capabilityName)
    }
    
    // تحديد نوع الأداة
    logicalTool, isLogical := el.sharedTools[toolName]
    if !isLogical {
        executionTool, exists := el.executionTools[toolName]
        if !exists {
            return nil, fmt.Errorf("tool not found: %s", toolName)
        }
        
        // تنفيذ في صندوق رمل معزول
        return el.executeInSandbox(ctx, agentID, executionTool, action, args)
    }
    
    // تنفيذ مشترك
    return el.executeShared(ctx, agentID, logicalTool, action, args)
}

func (el *ExecutionLayer) executeShared(
    ctx context.Context,
    agentID string,
    tool LogicalTool,
    action string,
    args map[string]interface{},
) (*Result, error) {
    
    // تنفيذ الأداة المشتركة
    switch action {
    case "read":
        return tool.Read(ctx, agentID, args)
    case "write":
        return tool.Write(ctx, agentID, args)
    case "execute":
        return tool.Execute(ctx, agentID, args)
    default:
        return nil, fmt.Errorf("action not supported: %s", action)
    }
}

func (el *ExecutionLayer) executeInSandbox(
    ctx context.Context,
    agentID string,
    tool ExecutionTool,
    action string,
    args map[string]interface{},
) (*Result, error) {
    
    // الحصول على صندوق الرمل للوكيل
    sandbox, err := el.sandboxManager.GetSandbox(agentID)
    if err != nil {
        return nil, fmt.Errorf("failed to get sandbox: %w", err)
    }
    
    // تنفيذ في الصندوق الرملي
    return sandbox.Execute(ctx, tool.Name(), action, args)
}
```

---

## نظام التفويضات المتكامل

### توسيع pkg/delegation

```go
// pkg/delegation/chain.go
type DelegationChain struct {
    ChainID      string
    OriginalDID  string
    Chain        []*DelegationRecord
    CreatedAt    time.Time
    ExpiresAt    time.Time
    Status       ChainStatus
}

type ChainStatus string

const (
    ChainStatusActive   ChainStatus = "active"
    ChainStatusExpired  ChainStatus = "expired"
    ChainStatusRevoked  ChainStatus = "revoked"
)

func NewDelegationChain(originalDID string, duration time.Duration) *DelegationChain {
    return &DelegationChain{
        ChainID:     generateChainID(),
        OriginalDID: originalDID,
        Chain:       make([]*DelegationRecord, 0),
        CreatedAt:   time.Now(),
        ExpiresAt:   time.Now().Add(duration),
        Status:      ChainStatusActive,
    }
}

func (dc *DelegationChain) AddDelegation(record *DelegationRecord) error {
    if dc.Status != ChainStatusActive {
        return fmt.Errorf("chain is not active")
    }
    
    if time.Now().After(dc.ExpiresAt) {
        dc.Status = ChainStatusExpired
        return fmt.Errorf("chain has expired")
    }
record)
    return nil
}

func (dc *DelegationChain) VerifyChain() error {
    // التحقق من صحة السلسلة
    for i, record := range dc.Chain {
        if time.Now().After(record.ExpiresAt) {
            return fmt.Errorf("delegation %d has expired", i)
        }
        
        // التحقق من التوقيع
        if err := verifySignature(record); err != nil {
            return fmt.Errorf("delegation %d has invalid signature: %w", i, err)
        }
    }
    
    return nil
}
```

```go
// pkg/delegation/conditional.go
type ConditionalDelegation struct {
    DelegationRecord *DelegationRecord
    Conditions      []Condition
}

type Condition struct {
    Type         string
    Value        interface{}
    Operator     string
    Required     bool
}

func (cd *ConditionalDelegation) Evaluate(ctx context.Context) (bool, error) {
    for _, condition := range cd.Conditions {
        satisfied, err := cd.evaluateCondition(ctx, condition)
        if err != nil {
            return false, err
        }
        
        if condition.Required && !satisfied {
            return false, nil
        }
    }
    
    return true, nil
}

func (cd *ConditionalDelegation) evaluateCondition(ctx context.Context, condition Condition) (bool, error) {
    switch condition.Type {
    case "time":
        return cd.evaluateTimeCondition(condition)
    case "location":
        return cd.evaluateLocationCondition(condition)
    case "resource":
        return cd.evaluateResourceCondition(ctx, condition)
    default:
        return false, fmt.Errorf("unknown condition type: %s", condition.Type)
    }
}
```

---

## نظام القدرات المتكامل

### توسيع pkg/capability

```go
// pkg/capability/token.go
type CapabilityToken struct {
    TokenID         string
    AgentID         string
    SessionID       string
    GrantedBy       string
    GrantedAt       time.Time
    ExpiresAt       time.Time
    Capabilities    map[string]*CapabilityGrant
    Conditions      map[string]interface{}
    Metadata        map[string]interface{}
    Status          TokenStatus
    DelegationChain *delegation.DelegationChain
    RevokedAt       *time.Time
    RevokedBy       string
}

type CapabilityGrant struct {
    CapabilityName string
    Actions        []string
    Resources      []string
    Constraints    map[string]interface{}
    GrantedAt      time.Time
    ExpiresAt      *time.Time
}

type TokenStatus string

const (
    TokenActive    TokenStatus = "active"
    TokenExpired   TokenStatus = "expired"
    TokenRevoked   TokenStatus = "revoked"
    TokenSuspended TokenStatus = "suspended"
)

func NewCapabilityToken(agentID, sessionID, grantedBy string, duration time.Duration) *CapabilityToken {
    return &CapabilityToken{
        TokenID:      generateTokenID(),
        AgentID:      agentID,
        SessionID:    sessionID,
        GrantedBy:    grantedBy,
        GrantedAt:    time.Now(),
        ExpiresAt:    time.Now().Add(duration),
        Capabilities: make(map[string]*CapabilityGrant),
        Conditions:   make(map[string]interface{}),
        Metadata:     make(map[string]interface{}),
        Status:       TokenActive,
    }
}

func (ct *CapabilityToken) AddCapability(grant *CapabilityGrant) {
    ct.Capabilities[grant.CapabilityName] = grant
}

func (ct *CapabilityToken) RemoveCapability(capabilityName string) {
    delete(ct.Capabilities, capabilityName)
}

func (ct *CapabilityToken) HasCapability(capabilityName string, action string) bool {
    grant, exists := ct.Capabilities[capabilityName]
    if !exists {
        return false
    }
    
    for _, allowedAction := range grant.Actions {
        if allowedAction == action {
            return true
        }
    }
    
    return false
}

func (ct *CapabilityToken) IsValid() bool {
    if ct.Status != TokenActive {
        return false
    }
    
    if time.Now().After(ct.ExpiresAt) {
        ct.Status = TokenExpired
        return false
    }
    
    return true
}
```

```go
// pkg/capability/governance_manager.go
type CapabilityGovernanceManager struct {
    tokens          map[string]*CapabilityToken
    delegationLog   []*DelegationRecord
    policyEngine    *policy.Engine
    sessionManager  *SessionManager
    eventBus        *eventbus.EventBus
    logger          *zap.Logger
    mu              sync.RWMutex
}

func NewCapabilityGovernanceManager(
    policyEngine *policy.Engine,
    sessionManager *SessionManager,
    eventBus *eventbus.EventBus,
    logger *zap.Logger,
) *CapabilityGovernanceManager {
    return &CapabilityGovernanceManager{
        tokens:         make(map[string]*CapabilityToken),
        delegationLog:  make([]*DelegationRecord, 0),
        policyEngine:   policyEngine,
        sessionManager: sessionManager,
        eventBus:       eventBus,
        logger:         logger,
    }
}

func (cgm *CapabilityGovernanceManager) GrantCapability(
    ctx context.Context,
    delegator string,
    delegatee string,
    capability CapabilityGrant,
    duration time.Duration,
) (*CapabilityToken, error) {
    
    // التحقق من الصلاحيات
    if !cgm.canDelegate(ctx, delegator, capability) {
        return nil, fmt.Errorf("delegator does not have permission to grant this capability")
    }
    
    // البحث عن رمز موجود
    var token *CapabilityToken
    for _, t := range cgm.tokens {
        if t.AgentID == delegatee && t.Status == TokenActive {
            token = t
            break
        }
    }
    
    // إنشاء رمز جديد إذا لم يوجد
    if token == nil {
        sessionID := cgm.getSessionID(ctx)
        token = NewCapabilityToken(delegatee, sessionID, delegator, duration)
        cgm.tokens[token.TokenID] = token
    }
    
    // إضافة الصلاحية
    token.AddCapability(&capability)
    
    // تسجيل التفويض
    cgm.logDelegation(delegator, delegatee, token.TokenID, []string{capability.CapabilityName})
    
    // نشر حدث
    cgm.publishEvent("capability_granted", map[string]interface{}{
        "token_id": token.TokenID,
        "delegator": delegator,
        "delegatee": delegatee,
        "capability": capability.CapabilityName,
    })
    
    return token, nil
}

func (cgm *CapabilityGovernanceManager) RevokeCapability(
    ctx context.Context,
    revoker string,
    tokenID string,
    capabilityName string,
    reason string,
) error {
    
    cgm.mu.Lock()
    defer cgm.mu.Unlock()
    
    token, exists := cgm.tokens[tokenID]
    if !exists {
        return fmt.Errorf("token not found: %s", tokenID)
    }
    
    // التحقق من الصلاحية
    if !cgm.canRevoke(ctx, revoker, token) {
        return fmt.Errorf("revoker does not have permission to revoke this token")
    }
    
    // إزالة الصلاحية
    token.RemoveCapability(capabilityName)
    
    // نشر حدث
    cgm.publishEvent("capability_revoked", map[string]interface{}{
        "token_id": tokenID,
        "revoker": revoker,
        "capability": capabilityName,
        "reason": reason,
    })
    
    return nil
}

func (cgm *CapabilityGovernanceManager) CheckCapability(
    ctx context.Context,
    agentID string,
    capabilityName string,
    action string,
) bool {
    
    cgm.mu.RLock()
    defer cgm.mu.RUnlock()
    
    // البحث عن رمز نشط للوكيل
    for _, token := range cgm.tokens {
        if token.AgentID == agentID && token.IsValid() {
            return token.HasCapability(capabilityName, action)
        }
    }
    
    return false
}

func (cgm *CapabilityGovernanceManager) canDelegate(ctx context.Context, delegator string, capability CapabilityGrant) bool {
    // التحقق من أن المفوض لديه الصلاحية المطلوبة
    // يمكن استخدام policy engine للتحقق
    return true
}

func (cgm *CapabilityGovernanceManager) canRevoke(ctx context.Context, revoker string, token *CapabilityToken) bool {
    // التحقق من أن المُلغي لديه الصلاحية المطلوبة
    // يمكن استخدام policy engine للتحقق
    return true
}

func (cgm *CapabilityGovernanceManager) getSessionID(ctx context.Context) string {
    // الحصول على معرف الجلسة من context
    return "session-123"
}

func (cgm *CapabilityGovernanceManager) logDelegation(delegator, delegatee, tokenID string, capabilities []string) {
    record := &DelegationRecord{
        RecordID:      generateRecordID(),
        Delegator:     delegator,
        Delegatee:     delegatee,
        TokenID:       tokenID,
        Capabilities:  capabilities,
        GrantedAt:     time.Now(),
    }
    
    cgm.delegationLog = append(cgm.delegationLog, record)
}

func (cgm *CapabilityGovernanceManager) publishEvent(eventType string, data map[string]interface{}) {
    event := map[string]interface{}{
        "type":      eventType,
        "timestamp": time.Now(),
        "data":      data,
    }
    
    cgm.eventBus.Publish(event)
}
```

---

## نظام إدارة الصلاحيات للعميل البشري

### HumanCapabilityManager

```go
// pkg/agent/human_capability_manager.go
type HumanCapabilityManager struct {
    delegationManager    *delegation.DelegationManager
    capabilityGovernance *CapabilityGovernanceManager
    sessionManager       *SessionManager
    eventBus             *eventbus.EventBus
    logger               *zap.Logger
    mu                   sync.RWMutex
}

type SessionCapabilityConfig struct {
    SessionID           string
    Mode                ControlMode
    SessionManagerAgent string
    AgentCapabilities   map[string]*AgentCapabilityConfig
    HumanOverride       bool
    AllowDynamicMod    bool
    ModificationHistory []*CapabilityModification
    CreatedAt           time.Time
    LastModified        time.Time
}

type ControlMode string

const (
    ModeAutomatic ControlMode = "automatic"
    ModeManual    ControlMode = "manual"
)

type AgentCapabilityConfig struct {
    AgentID         string
    Role            string
    Capabilities    []CapabilityGrant
    DelegationChain []*delegation.DelegationRecord
    Restrictions    map[string]interface{}
    LastUpdated     time.Time
    UpdatedBy       string
}

type CapabilityModification struct {
    ModificationID  string
    AgentID         string
    OldCapabilities []CapabilityGrant
    NewCapabilities []CapabilityGrant
    ModifiedBy      string
    Reason          string
    ModifiedAt      time.Time
}

func NewHumanCapabilityManager(
    delegationManager *delegation.DelegationManager,
    capabilityGovernance *CapabilityGovernanceManager,
    sessionManager *SessionManager,
    eventBus *eventbus.EventBus,
    logger *zap.Logger,
) *HumanCapabilityManager {
    return &HumanCapabilityManager{
        delegationManager:    delegationManager,
        capabilityGovernance: capabilityGovernance,
        sessionManager:       sessionManager,
        eventBus:             eventBus,
        logger:               logger,
    }
}

func (hcm *HumanCapabilityManager) CreateAutomaticSession(
    ctx context.Context,
    humanClientID string,
    agents []string,
    taskDescription string,
) (*SessionCapabilityConfig, error) {
    
    // 1. إنشاء الجلسة
    sessionID := generateSessionID()
    
    // 2. اختيار مدير الجلسة تلقائياً
    sessionManagerAgent := hcm.selectSessionManagerAgent(agents)
    
    // 3. تحليل المهام المطلوبة
    requiredCapabilities := hcm.analyzeTaskRequirements(taskDescription)
    
    // 4. توزيع الصلاحيات تلقائياً
    agentCapabilities := make(map[string]*AgentCapabilityConfig)
    for _, agentID := range agents {
        capabilities := hcm.assignCapabilitiesForAgent(
            agentID,
            requiredCapabilities,
            sessionManagerAgent,
        )
        
        agentCapabilities[agentID] = &AgentCapabilityConfig{
            AgentID:      agentID,
            Role:         hcm.determineRole(agentID, capabilities),
            Capabilities: capabilities,
            LastUpdated:  time.Now(),
            UpdatedBy:    sessionManagerAgent,
        }
    }
    
    // 5. إنشاء التكوين
    config := &SessionCapabilityConfig{
        SessionID:           sessionID,
        Mode:                ModeAutomatic,
        SessionManagerAgent: sessionManagerAgent,
        AgentCapabilities:   agentCapabilities,
        HumanOverride:       true,
        AllowDynamicMod:     true,
        CreatedAt:           time.Now(),
        LastModified:        time.Now(),
    }
    
    // 6. تطبيق الصلاحيات
    if err := hcm.applyCapabilities(ctx, config); err != nil {
        return nil, fmt.Errorf("failed to apply capabilities: %w", err)
    }
    
    // 7. نشر حدث
    hcm.publishEvent("automatic_session_created", map[string]interface{}{
        "session_id": sessionID,
        "session_manager": sessionManagerAgent,
        "agents": agents,
    })
    
    return config, nil
}

func (hcm *HumanCapabilityManager) CreateManualSession(
    ctx context.Context,
    humanClientID string,
    sessionManagerAgent string,
    agentCapabilities map[string][]CapabilityGrant,
) (*SessionCapabilityConfig, error) {
    
    // 1. إنشاء الجلسة
    sessionID := generateSessionID()
    
    // 2. بناء تكوين الوكلاء
    agentConfigs := make(map[string]*AgentCapabilityConfig)
    for agentID, capabilities := range agentCapabilities {
        agentConfigs[agentID] = &AgentCapabilityConfig{
            AgentID:      agentID,
            Role:         hcm.determineRole(agentID, capabilities),
            Capabilities: capabilities,
            LastUpdated:  time.Now(),
            UpdatedBy:    humanClientID,
        }
    }
    
    // 3. إنشاء التكوين
    config := &SessionCapabilityConfig{
        SessionID:           sessionID,
        Mode:                ModeManual,
        SessionManagerAgent: sessionManagerAgent,
        AgentCapabilities:   agentConfigs,
        HumanOverride:       true,
        AllowDynamicMod:     true,
        CreatedAt:           time.Now(),
        LastModified:        time.Now(),
    }
    
    // 4. تطبيق الصلاحيات
    if err := hcm.applyCapabilities(ctx, config); err != nil {
        return nil, fmt.Errorf("failed to apply capabilities: %w", err)
    }
    
    // 5. نشر حدث
    hcm.publishEvent("manual_session_created", map[string]interface{}{
        "session_id": sessionID,
        "session_manager": sessionManagerAgent,
        "agents": getKeys(agentCapabilities),
    })
    
    return config, nil
}

func (hcm *HumanCapabilityManager) ModifyAgentCapabilities(
    ctx context.Context,
    sessionID string,
    humanClientID string,
    agentID string,
    newCapabilities []CapabilityGrant,
    reason string,
) error {
    
    // 1. التحقق من صلاحية التعديل
    config, err := hcm.getSessionConfig(sessionID)
    if err != nil {
        return fmt.Errorf("session not found: %w", err)
    }
    
    if !config.AllowDynamicMod {
        return fmt.Errorf("dynamic modification not allowed for this session")
    }
    
    // 2. حفظ الصلاحيات القديمة
    oldCapabilities := config.AgentCapabilities[agentID].Capabilities
    
    // 3. تحديث الصلاحيات
    config.AgentCapabilities[agentID].Capabilities = newCapabilities
    config.AgentCapabilities[agentID].LastUpdated = time.Now()
    config.AgentCapabilities[agentID].UpdatedBy = humanClientID
    
    // 4. تسجيل التعديل
    modification := &CapabilityModification{
        ModificationID:  generateModificationID(),
        AgentID:         agentID,
        OldCapabilities: oldCapabilities,
        NewCapabilities: newCapabilities,
        ModifiedBy:      humanClientID,
        Reason:          reason,
        ModifiedAt:      time.Now(),
    }
    config.ModificationHistory = append(config.ModificationHistory, modification)
    config.LastModified = time.Now()
    
    // 5. تطبيق الصلاحيات الجديدة
    if err := hcm.applyCapabilities(ctx, config); err != nil {
        // التراجع في حالة الفشل
        config.AgentCapabilities[agentID].Capabilities = oldCapabilities
        return fmt.Errorf("failed to apply capabilities: %w", err)
    }
    
    // 6. نشر حدث
    hcm.publishEvent("capabilities_modified", map[string]interface{}{
        "session_id": sessionID,
        "agent_id": agentID,
        "modified_by": humanClientID,
        "reason": reason,
    })
    
    return nil
}

func (hcm *HumanCapabilityManager) applyCapabilities(
    ctx context.Context,
    config *SessionCapabilityConfig,
) error {
    
    for agentID, agentConfig := range config.AgentCapabilities {
        for _, grant := range agentConfig.Capabilities {
            // منح الصلاحية عبر نظام Capability Governance
            _, err := hcm.capabilityGovernance.GrantCapability(
                ctx,
                config.SessionManagerAgent,
                agentID,
                grant,
                24*time.Hour,
            )
            if err != nil {
                return fmt.Errorf("failed to grant capability: %w", err)
            }
        }
    }
    
    return nil
}
```

---

## التسلسل المنطقي للتنفيذ

### المرحلة 1: البنية الأساسية (الأسبوع 1-2)

#### 1.1 توسيع pkg/delegation
- إضافة `chain.go` للتفويضات المتسلسلة
- إضافة `conditional.go` للتفويضات المشروطة
- تحديث `advanced.go` لدعم التفويضات المتقدمة
- إضافة اختبارات شاملة

#### 1.2 توسيع pkg/capability
- إضافة `token.go` لCapability Tokens
- إضافة `governance_manager.go` لإدارة الصلاحيات
- تحديث `types.go` لإضافة الأنواع الجديدة
- تحديث `manager.go` للتكامل مع Governance Manager

#### 1.3 بناء طبقة الأدوات
- إنشاء `pkg/agent/tools/logical/` للأدوات المنطقية
  - `memory.go`
  - `skills.go`
  - `channels.go`
  - `registry.go`
  - `knowledge.go`
- إنشاء `pkg/agent/tools/execution/` للأدوات التنفيذية
  - `terminal.go`
  - `browser.go`
  - `filesystem.go`
  - `github.go`
  - `docker.go`
  - `http.go`
  - `database.go`
  - `email.go`
- إنشاء `pkg/agent/tools/registry.go` لTool Registry

### المرحلة 2: طبقة التنفيذ (الأسبوع 3-4)

#### 2.1 بناء Workspace Manager
- إنشاء `pkg/agent/execution/workspace_manager.go`
- إدارة مساحات العمل المنفصلة لكل وكيل
- حل تعارضات الملفات الذكي

#### 2.2 بناء Sandbox Manager
- إنشاء `pkg/agent/execution/sandbox_manager.go`
- إدارة الحاويات المعزولة لكل وكيل
- دعم Docker containers

#### 2.3 بناء Execution Layer
- إنشاء `pkg/agent/execution/execution_layer.go`
- تكامل مع Capability Governance
- دعم الأدوات المشتركة والمعزولة

### المرحلة 3: نظام إدارة الصلاحيات (الأسبوع 5-6)

#### 3.1 بناء HumanCapabilityManager
- إنشاء `pkg/agent/human_capability_manager.go`
- دعم الوضع الأوتوماتيكي
- دعم الوضع اليدوي
- دعم التعديل الديناميكي

#### 3.2 التكامل مع SessionManager
- تحديث `pkg/agent/unified/session_manager.go`
- إضافة HumanCapabilityManager
- إدارة الصلاحيات الديناميكية

#### 3.3 التكامل مع OrchestratorEngine
- تحديث `pkg/orchestrator/orchestrator_engine.go`
- إضافة HumanCapabilityManager
- إدارة الصلاحيات على مستوى المنسق

### المرحلة 4: واجهة المستخدم (الأسبوع 7)

#### 4.1 API Endpoints
- إنشاء نقاط نهاية لإنشاء الجلسات
- إنشاء نقاط نهاية لتعديل الصلاحيات
- إنشاء نقاط نهاية لمراقبة الصلاحيات

#### 4.2 واجهة المستخدم
- تصميم واجهة لإنشاء الجلسات
- تصميم واجهة لتعديل الصلاحيات
- تصميم واجهة لمراقبة الصلاحيات

### المرحلة 5: الاختبارات والتحسين (الأسبوع 8)

#### 5.1 اختبارات الوحدة
- اختبارات لكل مكون جديد
- اختبارات للتكامل بين المكونات
- اختبارات للأمان

#### 5.2 اختبارات التكامل
- اختبارات شاملة للنظام
- اختبارات للأداء
- اختبارات للتحمل

#### 5.3 التحسين
- تحسين الأداء
- تحسين الأمان
- تحسين الاستقرار

---

## خطة التنفيذ الكاملة

### الأسبوع 1: البنية الأساسية - الجزء 1
- اليوم 1-2: توسيع pkg/delegation
- اليوم 3-4: توسيع pkg/capability
- اليوم 5: اختبارات وتحسين

### الأسبوع 2: البنية الأساسية - الجزء 2
- اليوم 1-3: بناء الأدوات المنطقية
- اليوم 4-5: بناء الأدوات التنفيذية

### الأسبوع 3: طبقة التنفيذ - الجزء 1
- اليوم 1-2: بناء Workspace Manager
- اليوم 3-4: بناء Sandbox Manager
- اليوم 5: اختبارات

### الأسبوع 4: طبقة التنفيذ - الجزء 2
- اليوم 1-3: بناء Execution Layer
- اليوم 4-5: التكامل مع الأدوات

### الأسبوع 5: نظام إدارة الصلاحيات - الجزء 1
- اليوم 1-3: بناء HumanCapabilityManager
- اليوم 4-5: التكامل مع SessionManager

### الأسبوع 6: نظام إدارة الصلاحيات - الجزء 2
- اليوم 1-2: التكامل مع OrchestratorEngine
- اليوم 3-5: اختبارات شاملة

### الأسبوع 7: واجهة المستخدم
- اليوم 1-3: API Endpoints
- اليوم 4-5: واجهة المستخدم

### الأسبوع 8: الاختبارات والتحسين
- اليوم 1-2: اختبارات الوحدة
- اليوم 3-4: اختبارات التكامل
- اليوم 5: التحسين النهائي

---

## تقرير إصلاح الثغرات الحرجة والمشاكل المعمارية

### التحليل الشامل للثغرات الحرجة

تم إجراء تحليل شامل للملفات التالية:
- pkg/agent/unified/ (28 ملف)
- pkg/agent/ (الملفات الأخرى)
- pkg/providers/ (الموفرين)
- pkg/cache/ و pkg/metrics/
- pkg/session/ و pkg/eventbus/
- pkg/node/ (pkg/p2p غير موجود)
- pkg/network/ (الاتصال والشبكة)
- pkg/crypto/ (التشفير)
- pkg/identity/ (إثبات الهوية)

### الثغرات الحرجة المكتشفة والمصححة

#### 1. ثغرة دالة Hash غير آمنة في Cache (CRITICAL)
**الملف**: `pkg/cache/redis.go`
**المشكلة**: استخدام دالة hash بسيطة غير آمنة مع تعليق "in production use proper hashing"
**الخطر**: Hash collisions، cache keys قابلة للتنبؤ، cache poisoning محتمل
**الإصلاح**: استبدال الدالة بـ SHA-256 مع hex encoding
```go
// قبل الإصلاح:
func hashPrompt(prompt string) string {
    hash := 0
    for i, c := range prompt {
        hash += int(c) * (i + 1)
    }
    return fmt.Sprintf("%d", hash)
}

// بعد الإصلاح:
func hashPrompt(prompt string) string {
    hash := sha256.Sum256([]byte(prompt))
    return hex.EncodeToString(hash[:])
}
```

#### 2. Race Condition في ProviderRegistry (CRITICAL)
**الملف**: `pkg/providers/register.go`
**المشكلة**: Global registry يتم الوصول إليه بدون mutex protection
**الخطر**: Data races، crashes، inconsistent state
**الإصلاح**: إضافة sync.RWMutex وحماية جميع العمليات
```go
type ProviderRegistry struct {
    providers map[ProviderType]Provider
    mu        sync.RWMutex
}
```

#### 3. ثابت MaxAgents غير معرف (CRITICAL)
**الملف**: `pkg/session/skills.go`
**المشكلة**: مرجع لثابت MaxAgents غير معرف في الملف
**الخطر**: Compilation error، الكود لن يبني
**الإصلاح**: استخدام قيمة ثابتة مباشرة (100) بدلاً من الثابت غير الموجود

#### 4. Memory Leak في ChunkAssembler (HIGH)
**الملف**: `pkg/node/direct.go`
**المشكلة**: Cleanup goroutine قد لا يتوقف بشكل صحيح أثناء Close
**الخطر**: Memory leaks، goroutine leaks
**الإصلاح**: إضافة mutex protection و safe channel closing
```go
func (ca *ChunkAssembler) Close() error {
    ca.mu.Lock()
    defer ca.mu.Unlock()
    
    select {
    case <-ca.stopCh:
        return nil
    default:
        close(ca.stopCh)
        return nil
    }
}
```

#### 5. نمو DLQ غير محدود (HIGH)
**الملف**: `pkg/eventbus/dlq.go`
**المشكلة**: عدم وجود فحص حجم أثناء معالجة إعادة المحاولة
**الخطر**: Unbounded memory consumption
**الإصلاح**: إضافة فحص حجم قبل إضافة الإدخالات
```go
if len(remainingEntries) >= dlq.maxSize {
    continue
}
```

#### 6. دالة contains غير آمنة (MEDIUM)
**الملف**: `pkg/session/memory.go`
**المشكلة**: تنفيذ يدوي غير آمن لـ string contains
**الخطر**: Search manipulation، data leakage
**الإصلاح**: استخدام strings.Contains القياسية
```go
func contains(s, substr string) bool {
    return strings.Contains(s, substr)
}
```

#### 7. احتمال Deadlock في Election (HIGH)
**الملف**: `pkg/node/session_lifecycle.go`
**المشكلة**: locking معقد مع احتمال deadlock
**الخطر**: System hang، leader election failure
**الإصلاح**: إضافة proper cleanup عند context cancellation
```go
select {
case <-lm.ctx.Done():
    lm.electionMu.Lock()
    lm.inElection = false
    lm.electionMu.Unlock()
    return
case <-time.After(delay):
}
```

#### 8. Silent Error Swallowing (HIGH)
**الملفات**: `pkg/session/memory.go`, `pkg/session/journal.go`
**المشكلة**: استخدام `_ = json.Marshal()` يهمل الأخطاء
**الخطر**: Data corruption، silent failures
**الإصلاح**: معالجة جميع الأخطاء بشكل صحيح
```go
// قبل الإصلاح:
data, _ := json.Marshal(event)

// بعد الإصلاح:
data, err := json.Marshal(event)
if err != nil {
    return fmt.Errorf("failed to marshal event: %w", err)
}
```

#### 9. Import Validation ضعيفة (HIGH)
**الملف**: `pkg/session/container.go`
**المشكلة**: Import function lacks deep validation
**الخطر**: Malicious data injection، state corruption
**الإصلاح**: إضافة validation شامل للبيانات المستوردة
```go
// التحقق من صحة OwnerDID
if data.SessionContainer.OwnerDID == "" {
    return fmt.Errorf("معرف المالك فارغ في بيانات التصدير")
}

// التحقق من حدود الموارد
if len(data.State.Agents) > MaxAgentsInState {
    return fmt.Errorf("عدد الوكلاء يتجاوز الحد الأقصى")
}
```

#### 10. Type Assertion غير آمن (MEDIUM)
**الملف**: `pkg/session/memory.go`
**المشكلة**: Type assertions بدون ok check
**الخطر**: Panics، crashes
**الإصلاح**: إضافة safe type assertions
```go
// قبل الإصلاح:
if event.AgentDID != value.(string) {

// بعد الإصلاح:
if agentDID, ok := value.(string); ok {
    if event.AgentDID != agentDID {
        return false
    }
} else {
    return false
}
```

#### 11. Stop() Panic Risk في BootstrapManager (HIGH)
**الملف**: `pkg/network/bootstrap.go`
**المشكلة**: Stop() قد يسبب panic إذا تم استدعاؤه مرتين
**الخطر**: System crash، goroutine leaks
**الإصلاح**: إضافة mutex protection و safe channel closing
```go
func (bm *BootstrapManager) Stop() {
    bm.mu.Lock()
    defer bm.mu.Unlock()
    
    select {
    case <-bm.stopChan:
        return
    default:
        close(bm.stopChan)
    }
}
```

#### 12. Weak Passphrase Validation (MEDIUM)
**الملف**: `pkg/crypto/keystore.go`
**المشكلة**: عدم وجود تحقق من قوة passphrase
**الخطر**: Brute force attacks، weak encryption
**الإصلاح**: إضافة minimum length validation
```go
if len(passphrase) < 8 {
    return fmt.Errorf("passphrase must be at least 8 characters")
}
```

#### 13. Hash Length Check Missing (MEDIUM)
**الملف**: `pkg/crypto/pow.go`
**المشكلة**: checkDifficulty لا يتحقق من طول hash
**الخطر**: Index out of bounds، incorrect validation
**الإصلاح**: إضافة hash length validation
```go
if len(hash) < KeyLen {
    return false
}
```

#### 14. Race Condition in saveToDisk (HIGH)
**الملف**: `pkg/identity/revocation.go`
**المشكلة**: saveToDisk يقرأ من map مع RLock لكن يكتب من goroutines أخرى
**الخطر**: Data races، file corruption
**الإصلاح**: استخدام atomic write مع temp file
```go
tmpPath := c.diskPath + ".tmp"
if err := os.WriteFile(tmpPath, data, 0600); err != nil {
    return err
}
return os.Rename(tmpPath, c.diskPath)
```

#### 15. Input Validation Missing (MEDIUM)
**الملف**: `pkg/identity/manager.go`
**المشكلة**: CreateOrUpdateIdentity لا يتحقق من المدخلات
**الخطر**: Invalid state، data corruption
**الإصلاح**: إضافة input validation
```go
if did == "" {
    return nil, fmt.Errorf("DID cannot be empty")
}
if nodeID == "" {
    return nil, fmt.Errorf("nodeID cannot be empty")
}
```

### المشاكل المعمارية المكتشفة

#### 1. pkg/p2p غير موجود
**المشكلة**: الدليل pkg/p2p غير موجود لكن قد يتم الرجوع إليه
**التوصية**: إما إنشاء الدليل أو إزالة المراجع إليه

#### 2. معالجة الأخطاء غير متسقة
**المشكلة**: بعض الدوال تهمل الأخطاء بصمت
**التوصية**: توحيد معالجة الأخطاء عبر جميع الملفات

#### 3. ثوابت Hardcoded
**المشكلة**: قيم كثيرة hardcoded بدون configuration
**التوصية**: نقل الثوابت إلى ملف config مركزي

#### 4. احتمال Circular Dependencies
**المشكلة**: SessionContainer يرجع مكونات متعددة قد تخلق circular deps
**التوصية**: مراجعة بنية التبعيات

#### 5. عدم وجود Rate Limiting على بعض العمليات
**المشكلة**: بعض العمليات تفتقر rate limiting
**التوصية**: إضافة rate limiting للعمليات الحرجة

### ملخص الإصلاحات

تم إصلاح **16 ثغرة حرجة** و **5 مشاكل معمارية**:

**الثغرات الحرجة المصححة:**
1. ✅ Hash function آمنة في cache
2. ✅ Race condition في ProviderRegistry
3. ✅ Undefined MaxAgents في skills.go
4. ✅ Memory leak في ChunkAssembler
5. ✅ Unbounded DLQ growth
6. ✅ Insecure contains function
7. ✅ Election deadlock potential
8. ✅ Silent error swallowing في memory.go
9. ✅ Import validation في container.go
10. ✅ Silent error swallowing في journal.go
11. ✅ Type assertion safety في memory.go
12. ✅ Stop() panic risk في BootstrapManager
13. ✅ Weak passphrase validation في keystore
14. ✅ Hash length check missing في PoW
15. ✅ Race condition في saveToDisk
16. ✅ Input validation missing في identity manager

**المشاكل المعمارية المحددة:**
1. ⚠️ pkg/p2p غير موجود (يحتاج قرار)
2. ⚠️ معالجة الأخطاء غير متسقة (يحتاج توحيد)
3. ⚠️ ثوابت Hardcoded (يحتاج إعادة هيكلة)
4. ⚠️ احتمال Circular Dependencies (يحتاج مراجعة)
5. ⚠️ Rate Limiting مفقود (يحتاج إضافة)

### التوصيات للمستقبل

1. **الأمان**: إضافة security audit دوري
2. **الاختبار**: إضافة unit tests لجميع الإصلاحات
3. **المراقبة**: إضافة monitoring للعمليات الحرجة
4. **التوثيق**: تحديث التوثيق ليعكس الإصلاحات
5. **Code Review**: إنشاء process لـ code review صارم

---

## الاستنتاج

هذا التقرير الشامل يغطي:

1. **الملفات الموجودة**: تحليل شامل لكل الملفات الموجودة والمطلوب إضافته
2. **طبقة الأدوات الحقيقية**: 30+ أداة مقسمة إلى أدوات منطقية مشتركة وأدوات تنفيذية معزولة
3. **طبقة التنفيذ الحقيقية**: Workspace Manager و Sandbox Manager و Execution Layer
4. **نظام التفويضات المتكامل**: دعم التفويضات المتسلسلة والمشروطة والمؤقتة
5. **نظام القدرات المتكامل**: Capability Tokens و Governance Manager
6. **نظام إدارة الصلاحيات للعميل البشري**: أوضاع أوتوماتيكي ويدوي وتعديل ديناميكي
7. **التسلسل المنطقي للتنفيذ**: 8 أسابيع من التنفيذ المنظم
8. **خطة التنفيذ الكاملة**: خطة تفصيلية لكل أسبوع ويوم
9. **تقرير إصلاح الثغرات الحرجة والمشاكل المعمارية**: تحليل شامل وإصلاح 16 ثغرة حرجة و5 مشاكل معمارية

النظام النهائي سيكون:
- نظام تشغيل حقيقي للوكلاء والبشر
- يدعم عشرات الوكلاء المتزامنين بدون انتظار
- تعاون كامل مع صلاحيات ديناميكية
- أمان شامل مع تتبع كامل
- سرعة عالية مع كفاءة استخدام الموارد
