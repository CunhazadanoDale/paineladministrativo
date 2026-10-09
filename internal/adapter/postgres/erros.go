package postgres

import (
	"errors"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/lib/pq"
)

const (
	codigoViolacaoChaveEstrangeira = "23503"
	codigoViolacaoChaveUnica       = "23505"
)

func tratarErro(err error) error {
	if err == nil {
		return nil
	}

	var erroPostgres *pq.Error
	if errors.As(err, &erroPostgres) {
		switch erroPostgres.Code {
		case codigoViolacaoChaveEstrangeira:
			return domain.ErroValidacao("registro em uso por outros dados e não pode ser excluído")
		case codigoViolacaoChaveUnica:
			return domain.ErroConflito("já existe um registro com estes dados")
		}
	}

	return err
}
