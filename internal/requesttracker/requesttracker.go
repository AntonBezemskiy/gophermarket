// Этот пакет реализут мидлварь, которая будет добавлять каждый новый запрос к серверу в WaitGroup,
// таким образом перед завершением работы сервер обработает все уже поступившие к нему запросы.
// Мидлварь является одним из механизмов graceful shutdown.
package requesttracker

import (
	"context"
	"net/http"
	"sync"
)

var activeRequests sync.WaitGroup

func WithActiveRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		activeRequests.Add(1)
		defer activeRequests.Done()

		next.ServeHTTP(w, r)
	})
}

func WaitForActiveRequests(ctx context.Context) {
	activeRequests.Wait() // Block until all active requests are finished
}
