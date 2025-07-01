monorepo-controller: env ARGOCD_BINARY_NAME=argocd-monorepo-controller ./dist/argocd --loglevel trace --monorepo-repo-server-strict-tls=false --monorepo-repo-server-plaintext=false --monorepo-repo-server=127.0.0.1:8091
monorepo-repo-server: env ARGOCD_BINARY_NAME=argocd-monorepo-repo-server ARGOCD_GPG_ENABLED=false PATH="`pwd`/dist:$PATH" ./dist/argocd --loglevel debug
