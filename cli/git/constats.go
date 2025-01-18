package git

type conditionalOption string

const (
	Empty       conditionalOption = "empty"
	Uncommitted conditionalOption = "uncommitted"
	Unpushed    conditionalOption = "unpushed"
)
