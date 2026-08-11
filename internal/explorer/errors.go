package explorer

// ExitCodeError exits with the given exit code without printing an error
// message. It mirrors click's click.exceptions.Exit.
type ExitCodeError struct {
	Code int
}

// Error implements the error interface.
func (e *ExitCodeError) Error() string { return "" }
