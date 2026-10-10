package postgres

import (
	"errors"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

func TestTraduzirErroDoPostgres(t *testing.T) {
	casos := []struct {
		codigo    pqerror.Code
		sentinela error
		exclusao  string
		gravacao  string
	}{
		{pqerror.ForeignKeyViolation, domain.ErrValidacao, "registro em uso por outros dados e não pode ser excluído", "registro relacionado não encontrado"},
		{pqerror.UniqueViolation, domain.ErrConflito, "já existe um registro com estes dados", "já existe um registro com estes dados"},
		{pqerror.NotNullViolation, domain.ErrValidacao, "campo obrigatório não informado", "campo obrigatório não informado"},
		{pqerror.CheckViolation, domain.ErrValidacao, "dados fora das regras do cadastro", "dados fora das regras do cadastro"},
		{pqerror.StringDataRightTruncation, domain.ErrValidacao, "texto maior que o tamanho permitido", "texto maior que o tamanho permitido"},
		{pqerror.NumericValueOutOfRange, domain.ErrValidacao, "número fora do intervalo permitido", "número fora do intervalo permitido"},
		{pqerror.InvalidTextRepresentation, domain.ErrValidacao, "valor em formato inválido", "valor em formato inválido"},
	}

	for _, caso := range casos {
		t.Run(string(caso.codigo), func(t *testing.T) {
			original := &pq.Error{Code: caso.codigo}

			for _, traducao := range []struct {
				nome     string
				erro     error
				esperado string
			}{
				{"exclusão", tratarErro(original), caso.exclusao},
				{"gravação", tratarErroDeGravacao(original), caso.gravacao},
			} {
				if !errors.Is(traducao.erro, caso.sentinela) {
					t.Errorf("%s: erro %v, esperado sentinela %v", traducao.nome, traducao.erro, caso.sentinela)
				}
				if got := traducao.erro.Error(); got != caso.sentinela.Error()+": "+traducao.esperado {
					t.Errorf("%s: mensagem %q, esperado detalhe %q", traducao.nome, got, traducao.esperado)
				}
			}
		})
	}
}

func TestTraduzirErroPreservaOutrosErros(t *testing.T) {
	if tratarErroDeGravacao(nil) != nil {
		t.Error("nil deveria continuar nil")
	}

	desconhecido := &pq.Error{Code: "40001"}
	if err := tratarErroDeGravacao(desconhecido); err != desconhecido {
		t.Errorf("erro %v, esperado o original", err)
	}

	comum := errors.New("conexão recusada")
	if err := tratarErro(comum); err != comum {
		t.Errorf("erro %v, esperado o original", err)
	}
}
