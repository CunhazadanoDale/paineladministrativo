package importacao

import (
	"context"

	domainimportacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/importacao"
	"github.com/google/uuid"
)

type AlertasFiltro struct {
	Empresa    *uuid.UUID
	EmpresaIDs []uuid.UUID
}

type ConsultaRepository interface {
	AlocacoesAbertas(ctx context.Context, filtro AlertasFiltro) ([]domainimportacao.AlocacaoAberta, error)
	PedidosComFalta(ctx context.Context, filtro AlertasFiltro) ([]domainimportacao.PedidoSemImportacao, error)
	EmpresaDoPedido(ctx context.Context, pedidoID uuid.UUID) (*uuid.UUID, error)
	AtendimentosDoPedido(ctx context.Context, pedidoID uuid.UUID) ([]domainimportacao.AtendimentoDoPedido, error)
}
