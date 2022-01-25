package buildah

import (
	"github.com/GoogleContainerTools/skaffold/proto/v1"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
)

func containerfileNotFound(err error, artifact string) error {
	return sErrors.NewError(err,
		&proto.ActionableErr{
			Message: err.Error(),
			ErrCode: proto.StatusCode_BUILD_DOCKERFILE_NOT_FOUND,
		})
}
