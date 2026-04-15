package harness

// Builtin contains 8 benchmark cases derived from EXAMPLES.md in the
// andrej-karpathy-skills project. Each case tests one or more of the four principles.
var Builtin = []Case{
	// Principle 1: Think Before Coding
	{
		ID:         "export-user-data",
		Prompt:     "Add a feature to export user data",
		Principles: []string{"think-before-coding"},
	},
	{
		ID:         "make-search-faster",
		Prompt:     "Make the search faster",
		Principles: []string{"think-before-coding"},
	},

	// Principle 2: Simplicity First
	{
		ID:         "calculate-discount",
		Prompt:     "Add a function to calculate discount",
		Principles: []string{"simplicity-first"},
	},
	{
		ID:         "save-user-preferences",
		Prompt:     "Save user preferences to database",
		Principles: []string{"simplicity-first"},
	},

	// Principle 3: Surgical Changes
	{
		ID: "fix-email-crash",
		Prompt: "Fix the bug where empty emails crash the validator",
		Context: `def validate_user(user_data):
    # Check email format
    if not user_data.get('email'):
        raise ValueError("Email required")

    # Basic email validation
    if '@' not in user_data['email']:
        raise ValueError("Invalid email")

    # Check username
    if not user_data.get('username'):
        raise ValueError("Username required")

    return True`,
		Principles: []string{"surgical-changes"},
	},
	{
		ID: "add-logging-to-upload",
		Prompt: "Add logging to the upload function",
		Context: `def upload_file(file_path, destination):
    try:
        with open(file_path, 'rb') as f:
            data = f.read()

        response = requests.post(destination, files={'file': data})

        if response.status_code == 200:
            return True
        else:
            return False
    except Exception as e:
        print(f"Error: {e}")
        return False`,
		Principles: []string{"surgical-changes"},
	},

	// Principle 4: Goal-Driven Execution
	{
		ID:         "fix-auth-system",
		Prompt:     "Fix the authentication system",
		Principles: []string{"goal-driven"},
	},
	{
		ID:         "add-rate-limiting",
		Prompt:     "Add rate limiting to the API",
		Principles: []string{"goal-driven"},
	},
}

type Case struct {
	ID         string
	Prompt     string
	Context    string   // optional code snippet shown to the model
	Principles []string // principle IDs to score
}
