package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type relogioDeTeste struct {
	momento time.Time
}

func (r *relogioDeTeste) agora() time.Time {
	return r.momento
}

func requisicaoDe(ip string) *http.Request {
	requisicao := httptest.NewRequest(http.MethodPost, "/api/v1/usuarios/autenticar", nil)
	requisicao.RemoteAddr = ip + ":51000"

	return requisicao
}

func TestLimitadorBloqueiaCredencialDepoisDoLimite(t *testing.T) {
	relogio := &relogioDeTeste{momento: time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)}
	limitador := novoLimitadorComRelogio(relogio.agora)
	requisicao := requisicaoDe("10.0.0.1")

	for tentativa := 1; tentativa < limiteDeFalhasPorCredencial; tentativa++ {
		limitador.RegistrarFalha(requisicao, "ana@exemplo.com")
		if _, bloqueado := limitador.Bloqueio(requisicao, "ana@exemplo.com"); bloqueado {
			t.Fatalf("bloqueado após %d falhas, limite é %d", tentativa, limiteDeFalhasPorCredencial)
		}
	}

	limitador.RegistrarFalha(requisicao, "ANA@exemplo.com ")
	restante, bloqueado := limitador.Bloqueio(requisicao, "ana@exemplo.com")
	if !bloqueado || restante != janelaDeTentativas {
		t.Fatalf("bloqueio = %v por %v, esperado bloqueado por %v", bloqueado, restante, janelaDeTentativas)
	}

	if _, bloqueado := limitador.Bloqueio(requisicaoDe("10.0.0.2"), "ana@exemplo.com"); bloqueado {
		t.Error("o bloqueio de um IP trancou a conta para outro IP")
	}
	if _, bloqueado := limitador.Bloqueio(requisicao, "bruno@exemplo.com"); bloqueado {
		t.Error("o bloqueio de uma credencial afetou outro e-mail do mesmo IP")
	}

	relogio.momento = relogio.momento.Add(janelaDeTentativas + time.Second)
	if _, bloqueado := limitador.Bloqueio(requisicao, "ana@exemplo.com"); bloqueado {
		t.Error("bloqueio continuou depois da janela")
	}
}

func TestLimitadorBloqueiaOrigemQueTestaMuitosEmails(t *testing.T) {
	relogio := &relogioDeTeste{momento: time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)}
	limitador := novoLimitadorComRelogio(relogio.agora)
	requisicao := requisicaoDe("10.0.0.9")

	for indice := 0; indice < limiteDeFalhasPorOrigem; indice++ {
		limitador.RegistrarFalha(requisicao, string(rune('a'+indice%26))+"@exemplo.com"+string(rune('0'+indice/26)))
	}

	if _, bloqueado := limitador.Bloqueio(requisicao, "nova@exemplo.com"); !bloqueado {
		t.Error("origem com muitas falhas não foi bloqueada")
	}
}

func TestLimitadorZeraACredencialNoSucesso(t *testing.T) {
	relogio := &relogioDeTeste{momento: time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)}
	limitador := novoLimitadorComRelogio(relogio.agora)
	requisicao := requisicaoDe("10.0.0.1")

	for tentativa := 1; tentativa < limiteDeFalhasPorCredencial; tentativa++ {
		limitador.RegistrarFalha(requisicao, "ana@exemplo.com")
	}
	limitador.RegistrarSucesso(requisicao, "ana@exemplo.com")
	limitador.RegistrarFalha(requisicao, "ana@exemplo.com")

	if _, bloqueado := limitador.Bloqueio(requisicao, "ana@exemplo.com"); bloqueado {
		t.Error("falhas anteriores ao sucesso continuaram contando")
	}
}

func TestLimitadorReiniciaAJanelaDeFalhas(t *testing.T) {
	relogio := &relogioDeTeste{momento: time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)}
	limitador := novoLimitadorComRelogio(relogio.agora)
	requisicao := requisicaoDe("10.0.0.1")

	for tentativa := 1; tentativa < limiteDeFalhasPorCredencial; tentativa++ {
		limitador.RegistrarFalha(requisicao, "ana@exemplo.com")
	}

	relogio.momento = relogio.momento.Add(janelaDeTentativas + time.Second)
	limitador.RegistrarFalha(requisicao, "ana@exemplo.com")

	if _, bloqueado := limitador.Bloqueio(requisicao, "ana@exemplo.com"); bloqueado {
		t.Error("falhas de uma janela anterior somaram com a atual")
	}
}
