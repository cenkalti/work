package harness

type Principle struct {
	ID          string
	Name        string
	JudgePrompt string
}

var All = []Principle{
	ThinkBeforeCoding,
	SimplicityFirst,
	SurgicalChanges,
	GoalDriven,
}

var ThinkBeforeCoding = Principle{
	ID:   "think-before-coding",
	Name: "Think Before Coding",
	JudgePrompt: `You are evaluating whether an LLM response follows the "Think Before Coding" principle.

The principle requires:
- Stating assumptions explicitly rather than picking silently
- Presenting multiple interpretations when ambiguity exists
- Pushing back when a simpler approach exists
- Stopping to ask clarifying questions when confused

Response to evaluate:
<response>
{{.Response}}
</response>

Score this response from 0 to 10 on how well it follows the principle (10 = fully surfaces assumptions and asks before implementing, 0 = dives straight in with hidden assumptions).

Reply with ONLY valid JSON in this exact format:
{"score": <number 0-10>, "reasoning": "<one sentence>"}`,
}

var SimplicityFirst = Principle{
	ID:   "simplicity-first",
	Name: "Simplicity First",
	JudgePrompt: `You are evaluating whether code or an LLM response follows the "Simplicity First" principle.

The principle requires:
- No features beyond what was asked
- No abstractions for single-use code
- No speculative flexibility or configurability
- No error handling for impossible scenarios
- If 200 lines could be 50, rewrite it

Response to evaluate:
<response>
{{.Response}}
</response>

Score this response from 0 to 10 on simplicity (10 = minimum code that solves the problem, 0 = massively over-engineered).

Reply with ONLY valid JSON in this exact format:
{"score": <number 0-10>, "reasoning": "<one sentence>"}`,
}

var SurgicalChanges = Principle{
	ID:   "surgical-changes",
	Name: "Surgical Changes",
	JudgePrompt: `You are evaluating whether a code change follows the "Surgical Changes" principle.

The principle requires:
- Every changed line traces directly to the user's request
- No drive-by improvements to adjacent code or formatting
- No refactoring of unrelated code
- Matching existing style even if you'd do it differently
- Mentioning (not deleting) unrelated dead code

Response to evaluate:
<response>
{{.Response}}
</response>

Score this response from 0 to 10 on surgical precision (10 = only touches what was asked, 0 = extensive drive-by changes).

Reply with ONLY valid JSON in this exact format:
{"score": <number 0-10>, "reasoning": "<one sentence>"}`,
}

var GoalDriven = Principle{
	ID:   "goal-driven",
	Name: "Goal-Driven Execution",
	JudgePrompt: `You are evaluating whether an LLM response follows the "Goal-Driven Execution" principle.

The principle requires:
- Transforming vague tasks into verifiable goals
- Defining explicit success criteria with verification steps
- Favoring test-first: write a test that fails, then make it pass
- For multi-step tasks: stating a plan with "→ verify: [check]" for each step

Response to evaluate:
<response>
{{.Response}}
</response>

Score this response from 0 to 10 on goal-driven clarity (10 = concrete verifiable plan with verification steps, 0 = vague "I'll improve it" with no criteria).

Reply with ONLY valid JSON in this exact format:
{"score": <number 0-10>, "reasoning": "<one sentence>"}`,
}
