package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/docker/model-runner/cmd/cli/commands/completion"
	"github.com/docker/model-runner/cmd/cli/desktop"
	"github.com/docker/model-runner/cmd/cli/pkg/model"
	"github.com/docker/model-runner/cmd/cli/pkg/ollama"
	"github.com/docker/model-runner/cmd/cli/pkg/standalone"
	"github.com/docker/model-runner/cmd/cli/readline"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// readMultilineInput reads input from stdin, supporting both single-line and multiline input.
// For multiline input, it detects triple-quoted strings and shows continuation prompts.
func readMultilineInput(cmd *cobra.Command, scanner *bufio.Scanner) (string, error) {
	cmd.Print("> ")

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("error reading input: %v", err)
		}
		return "", fmt.Errorf("EOF")
	}

	line := scanner.Text()

	// Check if this is the start of a multiline input (triple quotes)
	tripleQuoteStart := ""
	if strings.HasPrefix(line, `"""`) {
		tripleQuoteStart = `"""`
	} else if strings.HasPrefix(line, "'''") {
		tripleQuoteStart = "'''"
	}

	// If no triple quotes, return a single line
	if tripleQuoteStart == "" {
		return line, nil
	}

	// Check if the triple quotes are closed on the same line
	restOfLine := line[3:]
	if strings.HasSuffix(restOfLine, tripleQuoteStart) && len(restOfLine) >= 3 {
		// Complete multiline string on single line
		return line, nil
	}

	// Start collecting multiline input
	var multilineInput strings.Builder
	multilineInput.WriteString(line)
	multilineInput.WriteString("\n")

	// Continue reading lines until we find the closing triple quotes
	for {
		cmd.Print("... ")

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return "", fmt.Errorf("error reading input: %v", err)
			}
			return "", fmt.Errorf("unclosed multiline input (EOF)")
		}

		line = scanner.Text()
		multilineInput.WriteString(line)

		// Check if this line contains the closing triple quotes
		if strings.Contains(line, tripleQuoteStart) {
			// Found closing quotes, we're done
			break
		}

		multilineInput.WriteString("\n")
	}

	return multilineInput.String(), nil
}

// generateInteractiveWithReadline provides an enhanced interactive mode with readline support
func generateInteractiveWithReadline(cmd *cobra.Command, desktopClient *desktop.Client, backend, model, apiKey string) error {
	usage := func() {
		fmt.Fprintln(os.Stderr, "Available Commands:")
		fmt.Fprintln(os.Stderr, "  /bye            Exit")
		fmt.Fprintln(os.Stderr, "  /?, /help       Help for a command")
		fmt.Fprintln(os.Stderr, "  /? shortcuts    Help for keyboard shortcuts")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, `Use """ to begin a multi-line message.`)
		fmt.Fprintln(os.Stderr, "")
	}

	usageShortcuts := func() {
		fmt.Fprintln(os.Stderr, "Available keyboard shortcuts:")
		fmt.Fprintln(os.Stderr, "  Ctrl + a            Move to the beginning of the line (Home)")
		fmt.Fprintln(os.Stderr, "  Ctrl + e            Move to the end of the line (End)")
		fmt.Fprintln(os.Stderr, "   Alt + b            Move back (left) one word")
		fmt.Fprintln(os.Stderr, "   Alt + f            Move forward (right) one word")
		fmt.Fprintln(os.Stderr, "  Ctrl + k            Delete the sentence after the cursor")
		fmt.Fprintln(os.Stderr, "  Ctrl + u            Delete the sentence before the cursor")
		fmt.Fprintln(os.Stderr, "  Ctrl + w            Delete the word before the cursor")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  Ctrl + l            Clear the screen")
		fmt.Fprintln(os.Stderr, "  Ctrl + c            Stop the model from responding")
		fmt.Fprintln(os.Stderr, "  Ctrl + d            Exit (/bye)")
		fmt.Fprintln(os.Stderr, "")
	}

	scanner, err := readline.New(readline.Prompt{
		Prompt:         "> ",
		AltPrompt:      "... ",
		Placeholder:    "Send a message (/? for help)",
		AltPlaceholder: `Use """ to end multi-line input`,
	})
	if err != nil {
		// Fall back to basic input mode if readline initialization fails
		return generateInteractiveBasic(cmd, desktopClient, backend, model, apiKey)
	}

	// Disable history if the environment variable is set
	if os.Getenv("DOCKER_MODEL_NOHISTORY") != "" {
		scanner.HistoryDisable()
	}

	fmt.Print(readline.StartBracketedPaste)
	defer fmt.Printf(readline.EndBracketedPaste)

	var sb strings.Builder
	var multiline bool

	for {
		line, err := scanner.Readline()
		switch {
		case errors.Is(err, io.EOF):
			fmt.Println()
			return nil
		case errors.Is(err, readline.ErrInterrupt):
			if line == "" {
				fmt.Println("\nUse Ctrl + d or /bye to exit.")
			}

			scanner.Prompt.UseAlt = false
			sb.Reset()

			continue
		case err != nil:
			return err
		}

		switch {
		case multiline:
			// check if there's a multiline terminating string
			before, ok := strings.CutSuffix(line, `"""`)
			sb.WriteString(before)
			if !ok {
				fmt.Fprintln(&sb)
				continue
			}

			multiline = false
			scanner.Prompt.UseAlt = false
		case strings.HasPrefix(line, `"""`):
			line := strings.TrimPrefix(line, `"""`)
			line, ok := strings.CutSuffix(line, `"""`)
			sb.WriteString(line)
			if !ok {
				// no multiline terminating string; need more input
				fmt.Fprintln(&sb)
				multiline = true
				scanner.Prompt.UseAlt = true
			}
		case scanner.Pasting:
			fmt.Fprintln(&sb, line)
			continue
		case strings.HasPrefix(line, "/help"), strings.HasPrefix(line, "/?"):
			args := strings.Fields(line)
			if len(args) > 1 {
				switch args[1] {
				case "shortcut", "shortcuts":
					usageShortcuts()
				default:
					usage()
				}
			} else {
				usage()
			}
			continue
		case strings.HasPrefix(line, "/exit"), strings.HasPrefix(line, "/bye"):
			return nil
		case strings.HasPrefix(line, "/"):
			fmt.Printf("Unknown command '%s'. Type /? for help\n", strings.Fields(line)[0])
			continue
		default:
			sb.WriteString(line)
		}

		if sb.Len() > 0 && !multiline {
			userInput := sb.String()

			if err := chatWithMarkdown(cmd, desktopClient, backend, model, userInput, apiKey); err != nil {
				cmd.PrintErr(handleClientError(err, "Failed to generate a response"))
				sb.Reset()
				continue
			}

			cmd.Println()
			sb.Reset()
		}
	}
}

// generateInteractiveBasic provides a basic interactive mode (fallback)
func generateInteractiveBasic(cmd *cobra.Command, desktopClient *desktop.Client, backend, model, apiKey string) error {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		userInput, err := readMultilineInput(cmd, scanner)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("Error reading input: %v", err)
		}

		if strings.ToLower(strings.TrimSpace(userInput)) == "/bye" {
			break
		}

		if strings.TrimSpace(userInput) == "" {
			continue
		}

		if err := chatWithMarkdown(cmd, desktopClient, backend, model, userInput, apiKey); err != nil {
			cmd.PrintErr(handleClientError(err, "Failed to generate a response"))
			continue
		}

		cmd.Println()
	}
	return nil
}

var (
	markdownRenderer *glamour.TermRenderer
	lastWidth        int
)

// StreamingMarkdownBuffer handles partial content and renders complete markdown blocks
type StreamingMarkdownBuffer struct {
	buffer       strings.Builder
	inCodeBlock  bool
	codeBlockEnd string // tracks the closing fence (``` or ```)
	lastFlush    int    // position of last flush
}

// NewStreamingMarkdownBuffer creates a new streaming markdown buffer
func NewStreamingMarkdownBuffer() *StreamingMarkdownBuffer {
	return &StreamingMarkdownBuffer{}
}

// AddContent adds new content to the buffer and returns any content that should be displayed
func (smb *StreamingMarkdownBuffer) AddContent(content string, shouldUseMarkdown bool) (string, error) {
	smb.buffer.WriteString(content)

	if !shouldUseMarkdown {
		// If not using markdown, just return the new content as-is
		result := content
		smb.lastFlush = smb.buffer.Len()
		return result, nil
	}

	return smb.processPartialMarkdown()
}

// processPartialMarkdown processes the buffer and returns content ready for display
func (smb *StreamingMarkdownBuffer) processPartialMarkdown() (string, error) {
	fullText := smb.buffer.String()

	// Look for code block start/end in the full text from our last position
	if !smb.inCodeBlock {
		// Check if we're entering a code block
		if idx := strings.Index(fullText[smb.lastFlush:], "```"); idx != -1 {
			// Found code block start
			beforeCodeBlock := fullText[smb.lastFlush : smb.lastFlush+idx]
			smb.inCodeBlock = true
			smb.codeBlockEnd = "```"

			// Stream everything before the code block as plain text
			smb.lastFlush = smb.lastFlush + idx
			return beforeCodeBlock, nil
		}

		// No code block found, stream all new content as plain text
		newContent := fullText[smb.lastFlush:]
		smb.lastFlush = smb.buffer.Len()
		return newContent, nil
	} else {
		// We're in a code block, look for the closing fence
		searchStart := smb.lastFlush
		if endIdx := strings.Index(fullText[searchStart:], smb.codeBlockEnd+"\n"); endIdx != -1 {
			// Found complete code block with newline after closing fence
			endPos := searchStart + endIdx + len(smb.codeBlockEnd) + 1
			codeBlockContent := fullText[smb.lastFlush:endPos]

			// Render the complete code block
			rendered, err := renderMarkdown(codeBlockContent)
			if err != nil {
				// Fallback to plain text
				smb.lastFlush = endPos
				smb.inCodeBlock = false
				return codeBlockContent, nil
			}

			smb.lastFlush = endPos
			smb.inCodeBlock = false
			return rendered, nil
		} else if endIdx := strings.Index(fullText[searchStart:], smb.codeBlockEnd); endIdx != -1 && searchStart+endIdx+len(smb.codeBlockEnd) == len(fullText) {
			// Found code block end at the very end of buffer (no trailing newline yet)
			endPos := searchStart + endIdx + len(smb.codeBlockEnd)
			codeBlockContent := fullText[smb.lastFlush:endPos]

			// Render the complete code block
			rendered, err := renderMarkdown(codeBlockContent)
			if err != nil {
				// Fallback to plain text
				smb.lastFlush = endPos
				smb.inCodeBlock = false
				return codeBlockContent, nil
			}

			smb.lastFlush = endPos
			smb.inCodeBlock = false
			return rendered, nil
		}

		// Still in code block, don't output anything until it's complete
		return "", nil
	}
}

// Flush renders and returns any remaining content in the buffer
func (smb *StreamingMarkdownBuffer) Flush(shouldUseMarkdown bool) (string, error) {
	fullText := smb.buffer.String()
	remainingContent := fullText[smb.lastFlush:]

	if remainingContent == "" {
		return "", nil
	}

	if !shouldUseMarkdown {
		return remainingContent, nil
	}

	rendered, err := renderMarkdown(remainingContent)
	if err != nil {
		return remainingContent, nil
	}

	return rendered, nil
}

// shouldUseMarkdown determines if Markdown rendering should be used based on color mode.
func shouldUseMarkdown(colorMode string) bool {
	supportsColor := func() bool {
		return !color.NoColor
	}

	switch colorMode {
	case "yes":
		return true
	case "no":
		return false
	case "auto":
		return supportsColor()
	default:
		return supportsColor()
	}
}

// getTerminalWidth returns the terminal width, with a fallback to 80.
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return 80
	}
	return width
}

// getMarkdownRenderer returns a Markdown renderer, recreating it if terminal width changed.
func getMarkdownRenderer() (*glamour.TermRenderer, error) {
	currentWidth := getTerminalWidth()

	// Recreate if width changed or renderer doesn't exist.
	if markdownRenderer == nil || currentWidth != lastWidth {
		r, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(currentWidth),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create markdown renderer: %w", err)
		}
		markdownRenderer = r
		lastWidth = currentWidth
	}

	return markdownRenderer, nil
}

func renderMarkdown(content string) (string, error) {
	r, err := getMarkdownRenderer()
	if err != nil {
		return "", fmt.Errorf("failed to create markdown renderer: %w", err)
	}

	rendered, err := r.Render(content)
	if err != nil {
		return "", fmt.Errorf("failed to render markdown: %w", err)
	}

	return rendered, nil
}

// runOllamaModel runs an Ollama model with the given prompt.
func runOllamaModel(cmd *cobra.Command, modelName, prompt string, detach bool) error {
	// Ensure the Ollama runner is available
	dockerClient, err := desktop.DockerClientForContext(dockerCLI, dockerCLI.CurrentContext())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Check if ollama container is running
	ctrID, _, _, err := standalone.FindOllamaContainer(cmd.Context(), dockerClient)
	if err != nil {
		return fmt.Errorf("failed to find Ollama container: %w", err)
	}
	if ctrID == "" {
		return fmt.Errorf("Ollama runner is not running. Please start it with 'docker model install-runner --ollama' or 'docker model start-runner --ollama'")
	}

	// Strip the ollama.com prefix
	strippedModelName := model.StripOllamaPrefix(modelName)
	ollamaClient := ollama.NewClient("http://localhost:" + fmt.Sprintf("%d", standalone.DefaultOllamaPort))

	// Handle detach mode - just ensure the model exists
	if detach {
		cmd.Printf("Ollama model %s loaded (detach mode)\n", strippedModelName)
		return nil
	}

	// If no prompt provided and stdin is not piped, run interactive mode
	if prompt == "" {
		fi, err := os.Stdin.Stat()
		if err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
			return fmt.Errorf("interactive mode not yet implemented for Ollama models")
		}
		
		// Read from stdin
		reader := bufio.NewReader(os.Stdin)
		input, err := io.ReadAll(reader)
		if err == nil {
			prompt = string(input)
		}
	}

	if prompt == "" {
		return fmt.Errorf("no prompt provided")
	}

	// Run the model with the prompt
	err = ollamaClient.Run(cmd.Context(), strippedModelName, prompt, func(response string) {
		cmd.Print(response)
	})
	if err != nil {
		return fmt.Errorf("failed to run Ollama model: %w", err)
	}

	cmd.Println()
	return nil
}

// chatWithMarkdown performs chat and streams the response with selective markdown rendering.
func chatWithMarkdown(cmd *cobra.Command, client *desktop.Client, backend, model, prompt, apiKey string) error {
	colorMode, _ := cmd.Flags().GetString("color")
	useMarkdown := shouldUseMarkdown(colorMode)
	debug, _ := cmd.Flags().GetBool("debug")

	if !useMarkdown {
		// Simple case: just stream as plain text
		return client.Chat(backend, model, prompt, apiKey, func(content string) {
			cmd.Print(content)
		}, false)
	}

	// For markdown: use streaming buffer to render code blocks as they complete
	markdownBuffer := NewStreamingMarkdownBuffer()

	err := client.Chat(backend, model, prompt, apiKey, func(content string) {
		// Use the streaming markdown buffer to intelligently render content
		rendered, err := markdownBuffer.AddContent(content, true)
		if err != nil {
			if debug {
				cmd.PrintErrln(err)
			}
			// Fallback to plain text on error
			cmd.Print(content)
		} else if rendered != "" {
			cmd.Print(rendered)
		}
	}, true)
	if err != nil {
		return err
	}

	// Flush any remaining content from the markdown buffer
	if remaining, flushErr := markdownBuffer.Flush(true); flushErr == nil && remaining != "" {
		cmd.Print(remaining)
	}

	return nil
}

func newRunCmd() *cobra.Command {
	var debug bool
	var backend string
	var ignoreRuntimeMemoryCheck bool
	var colorMode string
	var detach bool

	const cmdArgs = "MODEL [PROMPT]"
	c := &cobra.Command{
		Use:   "run " + cmdArgs,
		Short: "Run a model and interact with it using a submitted prompt or chat mode",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			switch colorMode {
			case "auto", "yes", "no":
				return nil
			default:
				return fmt.Errorf("--color must be one of: auto, yes, no (got %q)", colorMode)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate backend if specified
			if backend != "" {
				if err := validateBackend(backend); err != nil {
					return err
				}
			}

			// Validate API key for OpenAI backend
			apiKey, err := ensureAPIKey(backend)
			if err != nil {
				return err
			}

			modelName := args[0]
			prompt := ""
			argsLen := len(args)
			if argsLen > 1 {
				prompt = strings.Join(args[1:], " ")
			}

			// Check if this is an Ollama model
			if model.IsOllamaModel(modelName) {
				return runOllamaModel(cmd, modelName, prompt, detach)
			}

			// Only read from stdin if not in detach mode
			if !detach {
				fi, err := os.Stdin.Stat()
				if err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
					// Read all from stdin
					reader := bufio.NewReader(os.Stdin)
					input, err := io.ReadAll(reader)
					if err == nil {
						if prompt != "" {
							prompt += "\n\n"
						}

						prompt += string(input)
					}
				}
			}

			if debug {
				if prompt == "" {
					cmd.Printf("Running model %s\n", modelName)
				} else {
					cmd.Printf("Running model %s with prompt %s\n", modelName, prompt)
				}
			}

			if _, err := ensureStandaloneRunnerAvailable(cmd.Context(), cmd); err != nil {
				return fmt.Errorf("unable to initialize standalone model runner: %w", err)
			}

			// Do not validate the model in case of using OpenAI's backend, let OpenAI handle it
			if backend != "openai" {
				_, err := desktopClient.Inspect(modelName, false)
				if err != nil {
					if !errors.Is(err, desktop.ErrNotFound) {
						return handleNotRunningError(handleClientError(err, "Failed to inspect model"))
					}
					cmd.Println("Unable to find model '" + modelName + "' locally. Pulling from the server.")
					if err := pullModel(cmd, desktopClient, modelName, ignoreRuntimeMemoryCheck); err != nil {
						return err
					}
				}
			}

			// Handle --detach flag: just load the model without interaction
			if detach {
				// Make a minimal request to load the model into memory
				err := desktopClient.Chat(backend, modelName, "", apiKey, func(content string) {
					// Silently discard output in detach mode
				}, false)
				if err != nil {
					return handleClientError(err, "Failed to load model")
				}
				if debug {
					cmd.Printf("Model %s loaded successfully\n", modelName)
				}
				return nil
			}

			if prompt != "" {
				if err := chatWithMarkdown(cmd, desktopClient, backend, modelName, prompt, apiKey); err != nil {
					return handleClientError(err, "Failed to generate a response")
				}
				cmd.Println()
				return nil
			}

			// Use enhanced readline-based interactive mode when terminal is available
			if term.IsTerminal(int(os.Stdin.Fd())) {
				return generateInteractiveWithReadline(cmd, desktopClient, backend, modelName, apiKey)
			}

			// Fall back to basic mode if not a terminal
			return generateInteractiveBasic(cmd, desktopClient, backend, modelName, apiKey)
		},
		ValidArgsFunction: completion.ModelNames(getDesktopClient, 1),
	}
	c.Args = func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf(
				"'docker model run' requires at least 1 argument.\n\n" +
					"Usage:  docker model run " + cmdArgs + "\n\n" +
					"See 'docker model run --help' for more information",
			)
		}

		return nil
	}

	c.Flags().BoolVar(&debug, "debug", false, "Enable debug logging")
	c.Flags().StringVar(&backend, "backend", "", fmt.Sprintf("Specify the backend to use (%s)", ValidBackendsKeys()))
	c.Flags().MarkHidden("backend")
	c.Flags().BoolVar(&ignoreRuntimeMemoryCheck, "ignore-runtime-memory-check", false, "Do not block pull if estimated runtime memory for model exceeds system resources.")
	c.Flags().StringVar(&colorMode, "color", "auto", "Use colored output (auto|yes|no)")
	c.Flags().BoolVarP(&detach, "detach", "d", false, "Load the model in the background without interaction")

	return c
}
