package postgres

import (
	"errors"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

func tratarErro(err error) error {
	return traduzirErro(err, "registro em uso por outros dados e não pode ser excluído")
}

func tratarErroDeGravacao(err error) error {
	return traduzirErro(err, "registro relacionado não encontrado")
}

func traduzirErro(err error, chaveEstrangeira string) error {
	if err == nil {
		return nil
	}

	var erroPostgres *pq.Error
	if !errors.As(err, &erroPostgres) {
		return err
	}

	switch erroPostgres.Code {
	case pqerror.ForeignKeyViolation:
		return domain.ErroValidacao(chaveEstrangeira)
	case pqerror.UniqueViolation:
		return domain.ErroConflito("já existe um registro com estes dados")
	case pqerror.NotNullViolation:
		return domain.ErroValidacao("campo obrigatório não informado")
	case pqerror.CheckViolation:
		return domain.ErroValidacao("dados fora das regras do cadastro")
	case pqerror.StringDataRightTruncation:
		return domain.ErroValidacao("texto maior que o tamanho permitido")
	case pqerror.NumericValueOutOfRange:
		return domain.ErroValidacao("número fora do intervalo permitido")
	case pqerror.InvalidTextRepresentation:
		return domain.ErroValidacao("valor em formato inválido")
	default:
		return err
	}
}
