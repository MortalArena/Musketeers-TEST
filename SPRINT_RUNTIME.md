Good.

Now PROVE that every panel displays REAL runtime objects.

I do NOT want implementation summaries.

I want runtime proof.

Start the application.

Open the Dashboard.

Create ONE real Session.

Create ONE real Runtime Agent.

Execute ONE real task.

Then verify EVERY dashboard panel using REAL runtime data.

For EACH panel provide:

1. Which runtime object it reads from.
2. Which Go file.
3. Which function.
4. Screenshot or JSON response.
5. Explain why this is NOT mocked.

Verify these panels:

✓ Runtime Status
✓ Sessions
✓ Agents
✓ Thinking Engine
✓ Workflow
✓ Memory
✓ Providers
✓ EventBus
✓ Execution Trace

Then prove this specific statement:

"The selected LLM model became a UnifiedAgent inside the runtime."

Show:

Model Name

↓

Provider

↓

AgentRegistry entry

↓

AgentPool entry

↓

UnifiedAgent instance

↓

ThinkingEngine execution

↓

Workflow execution

↓

Memory update

↓

EventBus event

↓

Dashboard visualization

If ANY step cannot be proven with runtime evidence,
stop and fix the code.

Do NOT answer with "implemented".

Answer only with runtime evidence.


Show me the Agent ID displayed in the Dashboard.

Then show me where this exact Agent ID exists inside AgentRegistry.

Then show me where the same Agent ID exists inside AgentPool.

Then show me which UnifiedAgent owns it.

If these IDs differ, the Dashboard is fake.

