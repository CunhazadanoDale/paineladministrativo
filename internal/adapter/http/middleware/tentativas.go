package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	janelaDeTentativas          = 15 * time.Minute
	limiteDeFalhasPorCredencial = 5
	limiteDeFalhasPorOrigem     = 50
	limiteDeRegistros           = 10000
)

type registroDeFalhas struct {
	falhas       int
	inicio       time.Time
	bloqueadoAte time.Time
}

type LimitadorDeTentativas struct {
	mutex     sync.Mutex
	registros map[string]*registroDeFalhas
	agora     func() time.Time
}

func NovoLimitadorDeTentativas() *LimitadorDeTentativas {
	return novoLimitadorComRelogio(time.Now)
}

func novoLimitadorComRelogio(agora func() time.Time) *LimitadorDeTentativas {
	return &LimitadorDeTentativas{registros: map[string]*registroDeFalhas{}, agora: agora}
}

func (l *LimitadorDeTentativas) Bloqueio(r *http.Request, email string) (time.Duration, bool) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	agora := l.agora()
	var maior time.Duration
	for _, chave := range chavesDeTentativa(r, email) {
		registro, ok := l.registros[chave.valor]
		if !ok || !registro.bloqueadoAte.After(agora) {
			continue
		}
		if restante := registro.bloqueadoAte.Sub(agora); restante > maior {
			maior = restante
		}
	}

	return maior, maior > 0
}

func (l *LimitadorDeTentativas) RegistrarFalha(r *http.Request, email string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	agora := l.agora()
	if len(l.registros) >= limiteDeRegistros {
		l.descartarExpirados(agora)
	}

	for _, chave := range chavesDeTentativa(r, email) {
		registro, ok := l.registros[chave.valor]
		if !ok || agora.Sub(registro.inicio) > janelaDeTentativas {
			registro = &registroDeFalhas{inicio: agora}
			l.registros[chave.valor] = registro
		}

		registro.falhas++
		if registro.falhas >= chave.limite {
			registro.bloqueadoAte = agora.Add(janelaDeTentativas)
		}
	}
}

func (l *LimitadorDeTentativas) RegistrarSucesso(r *http.Request, email string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	delete(l.registros, chaveDeCredencial(r, email))
}

func (l *LimitadorDeTentativas) descartarExpirados(agora time.Time) {
	for chave, registro := range l.registros {
		if agora.Sub(registro.inicio) > janelaDeTentativas && !registro.bloqueadoAte.After(agora) {
			delete(l.registros, chave)
		}
	}
}

type chaveDeTentativa struct {
	valor  string
	limite int
}

func chavesDeTentativa(r *http.Request, email string) []chaveDeTentativa {
	return []chaveDeTentativa{
		{valor: chaveDeCredencial(r, email), limite: limiteDeFalhasPorCredencial},
		{valor: "origem|" + origemDaRequisicao(r), limite: limiteDeFalhasPorOrigem},
	}
}

func chaveDeCredencial(r *http.Request, email string) string {
	return "credencial|" + strings.ToLower(strings.TrimSpace(email)) + "|" + origemDaRequisicao(r)
}

func origemDaRequisicao(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}
