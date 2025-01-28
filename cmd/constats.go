package cmd

import "aww/internal/repository"

type conditionalOption string

const (
	Empty       conditionalOption = "empty"
	Uncommitted conditionalOption = "uncommitted"
	Unpushed    conditionalOption = "unpushed"
)

type projectAction func(project *repository.Project, groupAction *repository.GroupActions) error
