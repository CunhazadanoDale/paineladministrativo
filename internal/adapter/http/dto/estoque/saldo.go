package estoque

import (
	"time"

	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

type SaldoResponse struct {
	ProdutoID  uuid.UUID            `json:"produto_id"`
	Saldo      int                  `json:"saldo"`
	Reservado  int                  `json:"reservado"`
	Disponivel int                  `json:"disponivel"`
	NaoAlocado int                  `json:"nao_alocado"`
	Locais     []SaldoLocalResponse `json:"locais"`
}

func NovoSaldoResponse(saldo *domainestoque.SaldoProduto) SaldoResponse {
	return SaldoResponse{
		ProdutoID:  saldo.Produto.ID,
		Saldo:      saldo.Produto.Saldo.Quantidade(),
		Reservado:  saldo.Produto.Reservado.Quantidade(),
		Disponivel: saldo.Produto.Disponivel(),
		NaoAlocado: saldo.NaoAlocado,
		Locais:     NovoSaldoLocalResponses(saldo.Locais),
	}
}

type SaldoLocalResponse struct {
	ProdutoID     uuid.UUID `json:"produto_id"`
	ProdutoNome   string    `json:"produto_nome"`
	ProdutoCodigo *string   `json:"produto_codigo"`
	UnidadeMedida string    `json:"unidade_medida"`
	LocalID       uuid.UUID `json:"local_id"`
	LocalNome     string    `json:"local_nome"`
	EmpresaID     uuid.UUID `json:"empresa_id"`
	Quantidade    int       `json:"quantidade"`
	Reservado     int       `json:"reservado"`
	Disponivel    int       `json:"disponivel"`
	AtualizadoEm  time.Time `json:"atualizado_em"`
}

func NovoSaldoLocalResponses(itens []*domainestoque.SaldoLocalDetalhe) []SaldoLocalResponse {
	respostas := make([]SaldoLocalResponse, 0, len(itens))
	for _, item := range itens {
		respostas = append(respostas, SaldoLocalResponse{
			ProdutoID:     item.ProdutoID,
			ProdutoNome:   item.ProdutoNome,
			ProdutoCodigo: item.ProdutoCodigo,
			UnidadeMedida: string(item.UnidadeMedida),
			LocalID:       item.LocalID,
			LocalNome:     item.LocalNome,
			EmpresaID:     item.EmpresaID,
			Quantidade:    item.Quantidade,
			Reservado:     item.Reservado,
			Disponivel:    item.Disponivel(),
			AtualizadoEm:  item.AtualizadoEm,
		})
	}

	return respostas
}
