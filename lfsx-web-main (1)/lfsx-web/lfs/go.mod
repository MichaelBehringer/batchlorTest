module gitea.hama.de/LFS/lfsx-web/lfs

go 1.20

require (
	gitea.hama.de/LFS/go-logger v1.1.2
	gitea.hama.de/LFS/go-webserver v1.1.1
	gitea.hama.de/LFS/lfsx-web/controller v0.0.0
	github.com/go-chi/chi v1.5.4
)

require (
	github.com/justinas/nosurf v1.1.1 // indirect
	golang.org/x/sys v0.8.0 // indirect
)

replace gitea.hama.de/LFS/lfsx-web/controller => ../controller
