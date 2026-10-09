package estoque

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type resumoUseCaseFalso struct {
	erro   error
	resumo *domainestoque.ResumoEstoque
}

func (f *resumoUseCaseFalso) Resumo(_ context.Context) (*domainestoque.ResumoEstoque, error) {
	if f.erro != nil {
		return nil, f.erro
	}

	return f.resumo, nil
}

var _ portsin.ResumoUseCase = (*resumoUseCaseFalso)(nil)

func novoResumoDeTeste(t *testing.T) *domainestoque.ResumoEstoque {
	t.Helper()

	return &domainestoque.ResumoEstoque{
		Produtos: domainestoque.ResumoProdutos{
			TotalProdutos:        12,
			ProdutosAtivos:       10,
			ValorEstoqueCentavos: 1543200,
			ProdutosEstoqueBaixo: 3,
		},
		UltimosMovimentos: []*domainestoque.MovimentoResumo{
			{
				ID:          uuid.New(),
				ProdutoID:   uuid.New(),
				ProdutoNome: "Cimento CP II 50kg",
				Tipo:        domainestoque.TipoMovimentoSaida,
				Quantidade:  domainestoque.QuantidadeDe(6),
				SaldoApos:   domainestoque.SaldoDe(4),
				CriadoEm:    time.Now().UTC(),
			},
		},
	}
}

func TestConsultarResumoDevolveTotaisEMovimentos(t *testing.T) {
	usecase := &resumoUseCaseFalso{resumo: novoResumoDeTeste(t)}

	resposta := executaProtegido(t, NewResumoHandler(usecase).Consultar, http.MethodGet, "/api/v1/estoque/resumo", "", "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}

	var corpo struct {
		Dados struct {
			TotalProdutos        int64 `json:"total_produtos"`
			ProdutosAtivos       int64 `json:"produtos_ativos"`
			ValorEstoqueCentavos int64 `json:"valor_estoque_centavos"`
			ProdutosEstoqueBaixo int64 `json:"produtos_estoque_baixo"`
			UltimosMovimentos    []struct {
				ID          uuid.UUID `json:"id"`
				ProdutoID   uuid.UUID `json:"produto_id"`
				ProdutoNome string    `json:"produto_nome"`
				Tipo        string    `json:"tipo"`
				Quantidade  int       `json:"quantidade"`
				SaldoApos   int       `json:"saldo_apos"`
				CriadoEm    time.Time `json:"criado_em"`
			} `json:"ultimos_movimentos"`
		} `json:"dados"`
	}

	if err := json.Unmarshal(resposta.Body.Bytes(), &corpo); err != nil {
		t.Fatalf("não decodifiquei a resposta: %v", err)
	}

	if corpo.Dados.TotalProdutos != 12 || corpo.Dados.ProdutosAtivos != 10 {
		t.Errorf("totais = %d/%d, esperados 12/10", corpo.Dados.TotalProdutos, corpo.Dados.ProdutosAtivos)
	}
	if corpo.Dados.ValorEstoqueCentavos != 1543200 {
		t.Errorf("valor em estoque = %d, esperado 1543200 centavos", corpo.Dados.ValorEstoqueCentavos)
	}
	if corpo.Dados.ProdutosEstoqueBaixo != 3 {
		t.Errorf("produtos com estoque baixo = %d, esperado 3", corpo.Dados.ProdutosEstoqueBaixo)
	}
	if len(corpo.Dados.UltimosMovimentos) != 1 {
		t.Fatalf("últimos movimentos = %d, esperado 1", len(corpo.Dados.UltimosMovimentos))
	}

	movimento := corpo.Dados.UltimosMovimentos[0]
	if movimento.ProdutoNome != "Cimento CP II 50kg" {
		t.Errorf("nome do produto = %q", movimento.ProdutoNome)
	}
	if movimento.Tipo != "saida" || movimento.Quantidade != 6 || movimento.SaldoApos != 4 {
		t.Errorf("movimento = %+v, esperado saida com quantidade 6 e saldo 4", movimento)
	}
	if movimento.ID == uuid.Nil || movimento.ProdutoID == uuid.Nil {
		t.Error("movimento sem identificador ou sem produto")
	}
	if movimento.CriadoEm.IsZero() {
		t.Error("movimento sem data de criação")
	}
}

func TestConsultarResumoComFalhaResponde500(t *testing.T) {
	usecase := &resumoUseCaseFalso{erro: errors.New("falha no banco")}

	resposta := executaProtegido(t, NewResumoHandler(usecase).Consultar, http.MethodGet, "/api/v1/estoque/resumo", "", "")

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusInternalServerError)
}
