package autenticacao

import (
	"time"

	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var _ portsinautenticacao.TokenService = (*TokenService)(nil)

type reivindicacoes struct {
	VersaoSessao int `json:"ver"`
	jwt.RegisteredClaims
}

type TokenService struct {
	segredo   []byte
	expiracao time.Duration
}

func NovoTokenService(segredo string, expiracao time.Duration) *TokenService {
	return &TokenService{segredo: []byte(segredo), expiracao: expiracao}
}

func (s *TokenService) Gerar(usuarioID uuid.UUID, versaoSessao int) (string, time.Time, error) {
	if usuarioID == uuid.Nil || versaoSessao < 0 {
		return "", time.Time{}, errUsuarioInvalido
	}

	agora := time.Now().UTC()
	expiraEm := agora.Add(s.expiracao)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, reivindicacoes{
		VersaoSessao: versaoSessao,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   usuarioID.String(),
			IssuedAt:  jwt.NewNumericDate(agora),
			ExpiresAt: jwt.NewNumericDate(expiraEm),
		},
	})

	texto, err := token.SignedString(s.segredo)
	if err != nil {
		return "", time.Time{}, err
	}

	return texto, expiraEm, nil
}

func (s *TokenService) Validar(token string) (uuid.UUID, int, error) {
	if token == "" {
		return uuid.Nil, 0, errTokenInvalido
	}

	registro := &reivindicacoes{}

	if _, err := jwt.ParseWithClaims(token, registro, func(token *jwt.Token) (any, error) {
		return s.segredo, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired()); err != nil {
		return uuid.Nil, 0, errTokenInvalido
	}

	usuarioID, err := uuid.Parse(registro.Subject)
	if err != nil || registro.VersaoSessao < 0 {
		return uuid.Nil, 0, errTokenInvalido
	}

	return usuarioID, registro.VersaoSessao, nil
}
