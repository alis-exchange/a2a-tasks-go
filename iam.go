package tasks

import (
	"os"
	"strings"

	"go.alis.build/iam/v3"
)

const (
	roleTaskViewer = "roles/task.viewer"
	roleTaskOwner  = "roles/task.owner"

	taskPermissionGet    = "tasks.get"
	taskPermissionList   = "tasks.list"
	taskPermissionUpdate = "tasks.update"
)

func newTaskIAM(project string) (*iam.IAM, error) {
	if os.Getenv("ALIS_OS_PROJECT") == "" {
		project = strings.TrimSpace(project)
		if project == "" {
			project = "local"
		}
		if err := os.Setenv("ALIS_OS_PROJECT", project); err != nil {
			return nil, err
		}
	}
	return iam.New([]*iam.Role{
		{
			Name: roleTaskViewer,
			Permissions: []string{
				taskPermissionGet,
				taskPermissionList,
			},
			AllUsers: false,
		},
		{
			Name: roleTaskOwner,
			Permissions: []string{
				taskPermissionGet,
				taskPermissionList,
				taskPermissionUpdate,
			},
			AllUsers: false,
		},
	})
}
