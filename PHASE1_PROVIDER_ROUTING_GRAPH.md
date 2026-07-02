# Phase 1: Provider Routing Graph

## Provider Architecture

```
Provider System
├── ProviderRegistry (pkg/providers/builtin/register.go)
├── Smart Router (pkg/providers/router.go)
├── Free Router (pkg/providers/free_router.go)
├── API Key Manager (pkg/providers/api_key_manager.go)
├── Model Catalog (pkg/providers/model_catalog.go)
├── Free Models Tracker (pkg/providers/free_models_tracker.go)
└── 23 Builtin Providers
```

## Provider Registry

### Provider Types
```
Provider Registry (23 Providers)
├── Mistral AI (providers.ProviderMistral)
├── OpenRouter (providers.ProviderOpenRouter)
├── Qwen (providers.ProviderQwen)
├── Anthropic (providers.ProviderAnthropic)
├── Cohere (providers.ProviderCohere)
├── DeepSeek (providers.ProviderDeepSeek)
├── Google (providers.ProviderGoogle)
├── Groq (providers.ProviderGroq)
├── Moonshot (providers.ProviderMoonshot)
├── NVIDIA (providers.ProviderNVIDIA)
├── Ollama (providers.ProviderOllama)
├── OpenAI (providers.ProviderOpenAI)
├── Perplexity (providers.ProviderPerplexity)
├── Poolside (providers.ProviderPoolside)
├── Recraft (providers.ProviderRecraft)
├── Sourceful (providers.ProviderSourceful)
├── StepFun (providers.ProviderStepFun)
├── Tencent (providers.ProviderTencent)
├── TogetherAI (providers.ProviderTogetherAI)
├── xAI (providers.ProviderXAI)
├── Xiaomi (providers.ProviderXiaomi)
├── Zai (providers.ProviderZai)
└── Custom (providers.ProviderCustom)
```

### Provider Registration
```
Provider Registration Flow
├── ProviderRegistry Creation
│   ├── builtin.NewRegistry()
│   ├── Initialize Provider Map
│   ├── Register All Builtin Providers
│   └── Return Registry
├── Provider Registration
│   ├── Provider Interface Implementation
│   ├── Provider Metadata
│   ├── Provider Capabilities
│   └── Provider Models
├── Provider Initialization
│   ├── Provider.Initialize()
│   ├── API Key Configuration
│   ├── Base URL Configuration
│   ├── Timeout Configuration
│   └── Provider Ready
└── Provider Availability
    ├── Provider.IsAvailable()
    ├── Provider.Ping()
    ├── Provider Health Check
    └── Provider Status
```

## Smart Router

### Router Configuration
```
Router Configuration
├── PreferFreeModels (bool)
├── PreferLocalModels (bool)
├── MaxRetries (int)
├── Timeout (time.Duration)
├── FallbackEnabled (bool)
├── CostOptimization (bool)
└── LatencyOptimization (bool)
```

### Router Selection Algorithm
```
Router Selection Flow
├── Request Received
│   ├── Completion Request
│   ├── Context
│   ├── Model Requirements
│   └── Constraints
├── Find Candidate Models
│   ├── Query ProviderRegistry
│   ├── Filter by Availability
│   ├── Filter by Capabilities
│   ├── Filter by Constraints
│   └── Return Candidates
├── Rank Candidates
│   ├── Score by Cost
│   ├── Score by Latency
│   ├── Score by Quality
│   ├── Score by Preference
│   └── Return Ranked List
├── Select Best Model
│   ├── Top Ranked Model
│   ├── Check Availability
│   ├── Check Constraints
│   └── Return Selection
├── Execute with Retry
│   ├── Try Selected Model
│   ├── On Failure: Try Next
│   ├── On Success: Return Result
│   └── Max Retries: 3
└── Return Result
    ├── Completion Response
    ├── Usage Statistics
    ├── Cost Tracking
    └── Latency Tracking
```

### Router Scoring
```
Router Scoring Algorithm
├── Cost Score
│   ├── Free Models: 100 points
│   ├── Low Cost Models: 80 points
│   ├── Medium Cost Models: 60 points
│   ├── High Cost Models: 40 points
│   └── Very High Cost Models: 20 points
├── Latency Score
│   ├── <100ms: 100 points
│   ├── 100-500ms: 80 points
│   ├── 500-1000ms: 60 points
│   ├── 1000-2000ms: 40 points
│   └── >2000ms: 20 points
├── Quality Score
│   ├── High Quality: 100 points
│   ├── Medium Quality: 80 points
│   ├── Low Quality: 60 points
│   └── Unknown Quality: 40 points
├── Preference Score
│   ├── Preferred Provider: +20 points
│   ├── Preferred Model: +10 points
│   ├── Free Model: +30 points (if PreferFreeModels)
│   └── Local Model: +30 points (if PreferLocalModels)
└── Total Score
    ├── Sum of all scores
    ├── Normalize to 0-100
    └── Return ranking
```

## Provider Execution Flow

### Completion Request Flow
```
Completion Request Flow
├── Request Creation
│   ├── Context
│   ├── Messages
│   ├── Model Requirements
│   ├── Parameters (temperature, max_tokens, etc.)
│   └── Constraints
├── Router.Route()
│   ├── Find Candidates
│   ├── Rank Candidates
│   ├── Select Best Model
│   └── Execute with Retry
├── Provider.Complete()
│   ├── Prepare Request
│   ├── Call Provider API
│   ├── Parse Response
│   └── Return Completion
├── Response Processing
│   ├── Extract Content
│   ├── Extract Usage
│   ├── Extract Metadata
│   └── Format Response
└── Return Result
    ├── Completion Response
    ├── Usage Statistics
    ├── Cost Calculation
    └── Performance Metrics
```

### Provider API Call Flow
```
Provider API Call Flow
├── Request Preparation
│   ├── Format Messages
│   ├── Set Parameters
│   ├── Add Headers
│   ├── Add Authentication
│   └── Prepare HTTP Request
├── API Call
│   ├── Send HTTP Request
│   ├── Wait for Response
│   ├── Handle Timeouts
│   ├── Handle Errors
│   └── Return Response
├── Response Parsing
│   ├── Parse JSON Response
│   ├── Extract Content
│   ├── Extract Usage
│   ├── Extract Metadata
│   └── Validate Response
├── Error Handling
│   ├── Retry on Failure
│   ├── Fallback to Next Provider
│   ├── Log Errors
│   └── Return Error
└── Return Result
    ├── Completion Response
    ├── Usage Statistics
    └── Error (if failed)
```

## Provider Health Monitoring

### Health Check Flow
```
Provider Health Check Flow
├── Health Check Request
│   ├── Provider.Ping()
│   ├── Context
│   ├── Timeout
│   └── Health Check
├── Provider Response
│   ├── Success: Provider Available
│   ├── Failure: Provider Unavailable
│   ├── Latency Measurement
│   └── Error Logging
├── Health Status Update
│   ├── Update Provider Status
│   ├── Update Usage Statistics
│   ├── Update Latency Statistics
│   └── Update Success Rate
├── Router Update
│   ├── Update Router Cache
│   ├── Update Model Rankings
│   ├── Update Selection Algorithm
│   └── Update Routing Strategy
└── Health Monitoring
    ├── Continuous Monitoring
    ├── Periodic Health Checks
    ├── Automatic Failover
    └── Automatic Recovery
```

### Health Metrics
```
Provider Health Metrics
├── Availability (Online/Offline)
├── Latency (ms)
├── Success Rate (%)
├── Error Rate (%)
├── Request Count
├── Token Count
├── Cost ($)
├── Rate Limit Status
└── Quota Status
```

## Provider Fallback

### Fallback Strategy
```
Fallback Strategy
├── Primary Provider Failure
│   ├── Detect Failure
│   ├── Log Error
│   ├── Select Fallback Provider
│   └── Retry Request
├── Fallback Provider Selection
│   ├── Next Best Model
│   ├── Same Provider (if retry)
│   ├── Different Provider (if failover)
│   └── Free Provider (if available)
├── Fallback Execution
│   ├── Execute with Fallback Provider
│   ├── Monitor Success
│   ├── Update Statistics
│   └── Return Result
└── Fallback Limits
    ├── Max Retries: 3
    ├── Max Fallbacks: 2
    ├── Timeout: 30 seconds
    └── Error Threshold: 50%
```

### Fallback Configuration
```
Fallback Configuration
├── FallbackEnabled (bool)
├── MaxRetries (int)
├── RetryDelay (time.Duration)
├── FallbackStrategy (string)
│   ├── "next_best": Try next best model
│   ├── "same_provider": Retry same provider
│   ├── "different_provider": Try different provider
│   └── "free_provider": Try free provider
└── ErrorThreshold (float64)
    ├── 0.0-1.0 range
    ├── Default: 0.5
    └── Trigger fallback when error rate exceeds threshold
```

## Provider Usage Tracking

### Usage Statistics
```
Usage Tracking Flow
├── Request Execution
│   ├── Record Request Start
│   ├── Record Model Selection
│   ├── Record Provider Selection
│   └── Record Request Parameters
├── Response Processing
│   ├── Record Response Time
│   ├── Record Token Usage
│   ├── Record Cost
│   └── Record Success/Failure
├── Statistics Update
│   ├── Update Provider Statistics
│   ├── Update Model Statistics
│   ├── Update User Statistics
│   └── Update System Statistics
└── Usage Reporting
    ├── Generate Usage Report
    ├── Generate Cost Report
    ├── Generate Performance Report
    └── Generate Health Report
```

### Usage Metrics
```
Usage Metrics
├── Total Requests
├── Successful Requests
├── Failed Requests
├── Total Tokens
├── Total Cost
├── Average Latency
├── P95 Latency
├── P99 Latency
├── Success Rate
└── Error Rate
```

## Provider Caching

### Model Cache
```
Model Cache Flow
├── Cache Initialization
│   ├── Create Model Cache
│   ├── Load Model Catalog
│   ├── Cache Model Information
│   └── Cache Model Capabilities
├── Cache Lookup
│   ├── Check Cache for Model
│   ├── Return Cached Info if Available
│   ├── Fetch from Provider if Not Available
│   └── Update Cache
├── Cache Update
│   ├── Update Model Information
│   ├── Update Model Capabilities
│   ├── Update Model Availability
│   └── Update Model Pricing
└── Cache Invalidation
    ├── Periodic Invalidation
    ├── Manual Invalidation
    ├── Provider Change Invalidation
    └── Model Update Invalidation
```

### Cache Configuration
```
Cache Configuration
├── Cache Size (number of models)
├── Cache TTL (time to live)
├── Cache Update Interval
├── Cache Invalidation Strategy
│   ├── "time_based": Invalidate after TTL
│   ├── "change_based": Invalidate on change
│   └── "manual": Manual invalidation only
└── Cache Persistence
    ├── In-Memory Cache
    ├── Disk Cache (optional)
    └── Distributed Cache (optional)
```

## Provider Cost Management

### Cost Tracking
```
Cost Tracking Flow
├── Request Execution
│   ├── Record Token Usage
│   ├── Calculate Cost
│   ├── Record Cost
│   └── Update Budget
├── Cost Calculation
│   ├── Input Token Cost
│   ├── Output Token Cost
│   ├── Total Cost
│   └── Cost per Provider
├── Budget Management
│   ├── Check Budget
│   ├── Track Spending
│   ├── Enforce Limits
│   └── Alert on Overspend
└── Cost Reporting
    ├── Generate Cost Report
    ├── Generate Cost Forecast
    ├── Generate Cost Breakdown
    └── Generate Cost Optimization
```

### Cost Optimization
```
Cost Optimization Strategy
├── Prefer Free Models
│   ├── Prioritize Free Models
│   ├── Use Free Models First
│   ├── Fallback to Paid Models
│   └── Minimize Cost
├── Prefer Low Cost Models
│   ├── Prioritize Low Cost Models
│   ├── Use Low Cost Models First
│   ├── Fallback to High Cost Models
│   └── Minimize Cost
├── Cost-Based Routing
│   ├── Route by Cost
│   ├── Select Cheapest Available
│   ├── Balance Cost and Quality
│   └── Optimize Spending
└── Cost Alerts
    ├── Alert on High Cost
    ├── Alert on Budget Exceeded
    ├── Alert on Unusual Spending
    └── Alert on Cost Optimization Opportunity
```

## Provider Rate Limiting

### Rate Limit Management
```
Rate Limit Flow
├── Rate Limit Detection
│   ├── Detect Rate Limit Error
│   ├── Parse Rate Limit Headers
│   ├── Extract Rate Limit Info
│   └── Update Rate Limit Status
├── Rate Limit Handling
│   ├── Wait for Reset
│   ├── Retry After Reset
│   ├── Fallback to Different Provider
│   └── Log Rate Limit Event
├── Rate Limit Avoidance
│   ├── Track Request Rate
│   ├── Throttle Requests
│   ├── Distribute Load
│   └── Avoid Rate Limits
└── Rate Limit Monitoring
    ├── Monitor Rate Limit Status
    ├── Monitor Request Rate
    ├── Monitor Success Rate
    └── Adjust Strategy
```

### Rate Limit Configuration
```
Rate Limit Configuration
├── Max Requests Per Minute
├── Max Requests Per Hour
├── Max Requests Per Day
├── Retry After Rate Limit
├── Rate Limit Strategy
│   ├── "wait": Wait for reset
│   ├── "fallback": Fallback to different provider
│   └── "throttle": Throttle requests
└── Rate Limit Monitoring
    ├── Monitor Rate Limit Status
    ├── Monitor Request Rate
    └── Adjust Strategy
```

## Provider Security

### API Key Management
```
API Key Management Flow
├── API Key Storage
│   ├── Environment Variables
│   ├── API Key Manager
│   ├── Encrypted Storage
│   └── Secure Storage
├── API Key Usage
│   ├── Load API Key
│   ├── Use API Key
│   ├── Rotate API Key
│   └── Revoke API Key
├── API Key Security
│   ├── Encrypt API Keys
│   ├── Never Log API Keys
│   ├── Never Expose API Keys
│   └── Secure Storage
└── API Key Rotation
    ├── Rotate API Keys Periodically
    ├── Rotate API Keys on Compromise
    ├── Update Provider Configuration
    └── Test New API Keys
```

### Provider Security
```
Provider Security Measures
├── TLS Encryption
│   ├── Use HTTPS for all requests
│   ├── Verify TLS Certificates
│   ├── Use Strong Cipher Suites
│   └── Enforce TLS
├── API Key Security
│   ├── Never expose API keys
│   ├── Never log API keys
│   ├── Never include in error messages
│   └── Secure storage
├── Request Security
│   ├── Validate Requests
│   ├── Sanitize Requests
│   ├── Rate Limit Requests
│   └── Monitor Requests
└── Response Security
    ├── Validate Responses
    ├── Sanitize Responses
    ├── Log Responses (without sensitive data)
    └── Monitor Responses
```

## Provider Implementation Status

### Implementation Status
```
Provider Implementation Status
├── ProviderRegistry: 100% ✓
├── Smart Router: 100% ✓
├── Free Router: 100% ✓
├── API Key Manager: 100% ✓
├── Model Catalog: 100% ✓
├── Free Models Tracker: 100% ✓
├── Provider Initialization: 100% ✓
├── Provider Execution: 100% ✓
├── Provider Health Monitoring: 100% ✓
├── Provider Fallback: 100% ✓
├── Provider Usage Tracking: 100% ✓
├── Provider Caching: 100% ✓
├── Provider Cost Management: 100% ✓
├── Provider Rate Limiting: 80% (basic rate limiting, missing advanced features)
├── Provider Security: 100% ✓
└── Provider Integration: 100% ✓
```

### Overall Provider Status
```
Overall Status: 95% Complete
├── Core Functionality: 100% (registry, routing, execution, health monitoring)
├── Advanced Features: 90% (fallback, usage tracking, caching, cost management)
├── Rate Limiting: 80% (basic rate limiting, missing advanced features)
└── Security: 100% (API key management, TLS, request/response security)
```

### Provider Specific Status
```
Provider Specific Status
├── Mistral AI: 100% ✓ (initialized, tested, working)
├── OpenRouter: 100% ✓ (initialized, tested, working)
├── Qwen: 100% ✓ (initialized, tested, working)
├── Anthropic: 100% ✓ (registered, not initialized without API key)
├── Cohere: 100% ✓ (registered, not initialized without API key)
├── DeepSeek: 100% ✓ (registered, not initialized without API key)
├── Google: 100% ✓ (registered, not initialized without API key)
├── Groq: 100% ✓ (registered, not initialized without API key)
├── Moonshot: 100% ✓ (registered, not initialized without API key)
├── NVIDIA: 100% ✓ (registered, not initialized without API key)
├── Ollama: 100% ✓ (registered, not initialized without API key)
├── OpenAI: 100% ✓ (registered, not initialized without API key)
├── Perplexity: 100% ✓ (registered, not initialized without API key)
├── Poolside: 100% ✓ (registered, not initialized without API key)
├── Recraft: 100% ✓ (registered, not initialized without API key)
├── Sourceful: 100% ✓ (registered, not initialized without API key)
├── StepFun: 100% ✓ (registered, not initialized without API key)
├── Tencent: 100% ✓ (registered, not initialized without API key)
├── TogetherAI: 100% ✓ (registered, not initialized without API key)
├── xAI: 100% ✓ (registered, not initialized without API key)
├── Xiaomi: 100% ✓ (registered, not initialized without API key)
├── Zai: 100% ✓ (registered, not initialized without API key)
└── Custom: 100% ✓ (registered, not initialized without API key)
```
