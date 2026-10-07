package autenticacao

import (
	"time"

	portsinautenticacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/autenticacao"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var _ portsinautenticacao.TokenService = (*TokenService)(nil)

type TokenService struct {
	segredo   []byte
	expiracao time.Duration
}

func NovoTokenService(segredo string, expiracao time.Duration) *TokenService {
	return &TokenService{segredo: []byte(segredo), expiracao: expiracao}
}

func (s *TokenService) Gerar(usuarioID uuid.UUID) (string, time.Time, error) {
	if usuarioID == uuid.Nil {
		return "", time.Time{}, errUsuarioInvalido
	}

	agora := time.Now().UTC()
	expiraEm := agora.Add(s.expiracao)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   usuarioID.String(),
		IssuedAt:  jwt.NewNumericDate(agora),
		ExpiresAt: jwt.NewNumericDate(expiraEm),
	})

	texto, err := token.SignedString(s.segredo)
	if err != nil {
		return "", time.Time{}, err
	}

	return texto, expiraEm, nil
}

func (s *TokenService) Validar(token string) (uuid.UUID, error) {
	if token == "" {
		return uuid.Nil, errTokenInvalido
	}

	registro := &jwt.RegisteredClaims{}

	if _, err := jwt.ParseWithClaims(token, registro, func(token *jwt.Token) (any, error) {
		return s.segredo, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired()); err != nil {
		return uuid.Nil, errTokenInvalido
	}

	usuarioID, err := uuid.Parse(registro.Subject)
	if err != nil {
		return uuid.Nil, errTokenInvalido
	}

	return usuarioID, nil
}
