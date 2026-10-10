package autenticacao

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestTokenGeradoEVaido(t *testing.T) {
	servico := NovoTokenService("segredo-do-teste", time.Hour)
	usuarioID := uuid.New()

	token, expiraEm, err := servico.Gerar(usuarioID, 3)
	if err != nil {
		t.Fatalf("geração do token falhou: %v", err)
	}
	if token == "" {
		t.Fatal("geração devolveu token vazio")
	}
	if !expiraEm.After(time.Now().UTC()) {
		t.Errorf("expiração %v deveria estar no futuro", expiraEm)
	}

	validado, versao, err := servico.Validar(token)
	if err != nil {
		t.Fatalf("validação do token falhou: %v", err)
	}
	if validado != usuarioID {
		t.Errorf("usuário validado %q, esperado %q", validado, usuarioID)
	}
	if versao != 3 {
		t.Errorf("versão da sessão %d, esperada 3", versao)
	}
}

func TestGerarRejeitaUsuarioInvalido(t *testing.T) {
	servico := NovoTokenService("segredo-do-teste", time.Hour)

	if _, _, err := servico.Gerar(uuid.Nil, 0); !errors.Is(err, errUsuarioInvalido) {
		t.Errorf("erro %v, esperado usuário inválido", err)
	}
}

func TestValidarRejeitaSegredoDiferente(t *testing.T) {
	gerador := NovoTokenService("segredo-um", time.Hour)
	validador := NovoTokenService("segredo-dois", time.Hour)

	token, _, err := gerador.Gerar(uuid.New(), 0)
	if err != nil {
		t.Fatalf("geração do token falhou: %v", err)
	}

	if _, _, err := validador.Validar(token); !errors.Is(err, errTokenInvalido) {
		t.Errorf("erro %v, esperado token inválido", err)
	}
}

func TestValidarRejeitaTokenExpirado(t *testing.T) {
	servico := NovoTokenService("segredo-do-teste", -time.Minute)

	token, _, err := servico.Gerar(uuid.New(), 0)
	if err != nil {
		t.Fatalf("geração do token falhou: %v", err)
	}

	if _, _, err := servico.Validar(token); !errors.Is(err, errTokenInvalido) {
		t.Errorf("erro %v, esperado token inválido", err)
	}
}

func TestValidarRejeitaTokenSemAssinaturaPermitida(t *testing.T) {
	servico := NovoTokenService("segredo-do-teste", time.Hour)

	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.RegisteredClaims{
		Subject:   uuid.New().String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
	})

	texto, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("assinatura sem criptografia falhou: %v", err)
	}

	if _, _, err := servico.Validar(texto); !errors.Is(err, errTokenInvalido) {
		t.Errorf("erro %v, esperado token inválido", err)
	}
}

func TestValidarRejeitaTokenVazioOuSujo(t *testing.T) {
	servico := NovoTokenService("segredo-do-teste", time.Hour)

	casos := []string{"", "abc", "Bearer abc", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ4In0.assinatura"}

	for _, caso := range casos {
		if _, _, err := servico.Validar(caso); !errors.Is(err, errTokenInvalido) {
			t.Errorf("token %q devolveu erro %v, esperado token inválido", caso, err)
		}
	}
}

func TestGerarRejeitaVersaoNegativa(t *testing.T) {
	servico := NovoTokenService("segredo-do-teste", time.Hour)

	if _, _, err := servico.Gerar(uuid.New(), -1); !errors.Is(err, errUsuarioInvalido) {
		t.Errorf("erro %v, esperado usuário inválido", err)
	}
}

func TestValidarTokenSemVersaoAssumeVersaoZero(t *testing.T) {
	servico := NovoTokenService("segredo-do-teste", time.Hour)
	usuarioID := uuid.New()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   usuarioID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
	})

	texto, err := token.SignedString([]byte("segredo-do-teste"))
	if err != nil {
		t.Fatalf("assinatura falhou: %v", err)
	}

	validado, versao, err := servico.Validar(texto)
	if err != nil {
		t.Fatalf("validação falhou: %v", err)
	}
	if validado != usuarioID || versao != 0 {
		t.Errorf("validado %q versão %d, esperado %q versão 0", validado, versao, usuarioID)
	}
}
