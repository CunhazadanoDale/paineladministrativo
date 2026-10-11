package estoque

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	"github.com/google/uuid"
)

func TestTransferirRespondeCreatedERepassaLocais(t *testing.T) {
	movimento := novoMovimentoDeTeste(t)
	origem, destino := uuid.New(), uuid.New()
	movimento.Tipo = domainestoque.TipoMovimentoTransferencia
	movimento.LocalID = &origem
	movimento.LocalDestinoID = &destino
	saldoOrigem := domainestoque.SaldoDe(6)
	movimento.SaldoLocalApos = &saldoOrigem
	usecase := &movimentoUseCaseFalso{movimento: movimento}
	id := uuid.New()

	resposta := executaProtegido(t, NewMovimentoHandler(usecase).Transferir, http.MethodPost,
		"/api/v1/produtos/"+id.String()+"/transferencias", id.String(),
		`{"local_origem_id":"`+origem.String()+`","local_destino_id":"`+destino.String()+`","quantidade":4,"observacao":"abastecer a loja"}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusCreated, resposta.Body.String())
	}
	input := usecase.inputTransferir
	if input.ProdutoID != id || input.LocalOrigemID != origem || input.LocalDestinoID != destino || input.Quantidade != 4 || input.UsuarioID == uuid.Nil {
		t.Errorf("input repassado = %+v", input)
	}

	var envelope dto.Resposta[estoquedto.MovimentoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	dados := envelope.Dados
	if dados.Tipo != "transferencia" || *dados.LocalID != origem || *dados.LocalDestinoID != destino || *dados.SaldoLocalApos != 6 {
		t.Errorf("movimento devolvido = %+v", dados)
	}
}

func TestTransferirEntreEmpresasResponde400(t *testing.T) {
	usecase := &movimentoUseCaseFalso{erro: domain.ErroValidacao("transferência só entre locais da mesma empresa")}
	id := uuid.New()

	resposta := executaProtegido(t, NewMovimentoHandler(usecase).Transferir, http.MethodPost,
		"/api/v1/produtos/"+id.String()+"/transferencias", id.String(), `{"quantidade":1}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
}

func TestSaldoDevolveTotaisELocais(t *testing.T) {
	produto := novoProdutoDeTeste(t)
	produto.Saldo = domainestoque.SaldoDe(12)
	produto.Reservado = domainestoque.SaldoDe(2)
	localID := uuid.New()
	usecase := &movimentoUseCaseFalso{saldo: &domainestoque.SaldoProduto{
		Produto:    produto,
		NaoAlocado: 4,
		Locais: []*domainestoque.SaldoLocalDetalhe{{
			SaldoLocal: domainestoque.SaldoLocal{ProdutoID: produto.ID, LocalID: localID, Quantidade: 8, Reservado: 2},
			LocalNome:  "Galpão",
			EmpresaID:  uuid.New(),
		}},
	}}

	resposta := executaProtegido(t, NewMovimentoHandler(usecase).Saldo, http.MethodGet,
		"/api/v1/produtos/"+produto.ID.String()+"/saldo", produto.ID.String(), "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}

	var envelope dto.Resposta[estoquedto.SaldoResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	dados := envelope.Dados
	if dados.ProdutoID != produto.ID || dados.Saldo != 12 || dados.Reservado != 2 || dados.Disponivel != 10 || dados.NaoAlocado != 4 {
		t.Errorf("totais devolvidos = %+v", dados)
	}
	if len(dados.Locais) != 1 || dados.Locais[0].LocalID != localID || dados.Locais[0].Disponivel != 6 {
		t.Errorf("locais devolvidos = %+v", dados.Locais)
	}
}

func TestListarLotesRepassaFiltrosEDevolveCusto(t *testing.T) {
	localID := uuid.New()
	custo := domainestoque.CustoDe(2590)
	usecase := &movimentoUseCaseFalso{lotes: []*domainestoque.Lote{
		{ID: uuid.New(), LocalID: localID, Custo: &custo, QuantidadeInicial: 10, QuantidadeAtual: 4, EntradaEm: time.Now().UTC()},
		{ID: uuid.New(), LocalID: localID, QuantidadeInicial: 5, QuantidadeAtual: 5, EntradaEm: time.Now().UTC()},
	}}
	id := uuid.New()

	resposta := executaProtegido(t, NewMovimentoHandler(usecase).ListarLotes, http.MethodGet,
		"/api/v1/produtos/"+id.String()+"/lotes?local="+localID.String()+"&com_saldo=true", id.String(), "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	input := usecase.inputLotes
	if input.ProdutoID != id || input.LocalID == nil || *input.LocalID != localID || !input.ComSaldo {
		t.Errorf("input repassado = %+v", input)
	}

	var pagina dto.Paginado[estoquedto.LoteResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &pagina); err != nil {
		t.Fatalf("corpo não é uma página válida: %v", err)
	}
	if len(pagina.Dados) != 2 || pagina.Dados[0].CustoUnitarioCentavos == nil || *pagina.Dados[0].CustoUnitarioCentavos != 2590 || pagina.Dados[1].CustoUnitarioCentavos != nil {
		t.Errorf("lotes devolvidos = %+v", pagina.Dados)
	}
}

func TestListarLotesSemPerfilResponde403(t *testing.T) {
	usecase := &movimentoUseCaseFalso{erro: domain.ErroPermissao("perfil sem permissão para consultar o custo do estoque")}
	id := uuid.New()

	resposta := executaProtegido(t, NewMovimentoHandler(usecase).ListarLotes, http.MethodGet,
		"/api/v1/produtos/"+id.String()+"/lotes", id.String(), "")

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusForbidden)
}

func TestListarMovimentosRepassaLocal(t *testing.T) {
	usecase := &movimentoUseCaseFalso{}
	id, localID := uuid.New(), uuid.New()

	resposta := executaProtegido(t, NewMovimentoHandler(usecase).Listar, http.MethodGet,
		"/api/v1/produtos/"+id.String()+"/movimentos?local="+localID.String(), id.String(), "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if usecase.inputListar.LocalID == nil || *usecase.inputListar.LocalID != localID || usecase.inputListar.UsuarioID == uuid.Nil {
		t.Errorf("input repassado = %+v", usecase.inputListar)
	}
}
