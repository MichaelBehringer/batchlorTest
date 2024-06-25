package vnc

import (
	"io"

	"gitea.hama.de/LFS/lfsx-web/controller/internal/guacamole"
)

type Guacamole struct {
	Used bool

	Stream *guacamole.Stream

	Writer io.Writer
}
