package estoque

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	estoquedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/estoque"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainestoque "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/estoque"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/estoque"
	"github.com/google/uuid"
)

type localUseCaseFalso struct {
	erro          error
	local         *domainestoque.Local
	saldos        []*domainestoque.SaldoLocalDetalhe
	inputCriar    portsin.CriarLocalInput
	inputListar   portsin.ListarLocaisInput
	inputRenomear portsin.RenomearLocalInput
	inputSaldos   portsin.ListarSaldosDoLocalInput
	desativado    uuid.UUID
	removido      uuid.UUID
}

func (f *localUseCaseFalso) Criar(_ context.Context, input portsin.CriarLocalInput) (uuid.UUID, error) {
	if f.erro != nil {
		return uuid.Nil, f.erro
	}
	f.inputCriar = input

	return f.local.ID, nil
}

func (f *localUseCaseFalso) Obter(_ context.Context, _, _ uuid.UUID) (*domainestoque.Local, error) {
	return f.local, f.erro
}

func (f *localUseCaseFalso) Listar(_ context.Context, input portsin.ListarLocaisInput) ([]*domainestoque.Local, error) {
	if f.erro != nil {
		return nil, f.erro
	}
	f.inputListar = input

	return []*domainestoque.Local{f.local}, nil
}

func (f *localUseCaseFalso) Renomear(_ context.Context, input portsin.RenomearLocalInput) (*domainestoque.Local, error) {
	if f.erro != nil {
		return nil, f.erro
	}
	f.inputRenomear = input

	return f.local, nil
}

func (f *localUseCaseFalso) Ativar(_ context.Context, _, _ uuid.UUID) (*domainestoque.Local, error) {
	return f.local, f.erro
}

func (f *localUseCaseFalso) Desativar(_ context.Context, localID, _ uuid.UUID) (*domainestoque.Local, error) {
	if f.erro != nil {
		return nil, f.erro
	}
	f.desativado = localID

	return f.local, nil
}

func (f *localUseCaseFalso) Remover(_ context.Context, localID, _ uuid.UUID) error {
	if f.erro != nil {
		return f.erro
	}
	f.removido = localID

	return nil
}

func (f *localUseCaseFalso) ListarSaldos(_ context.Context, input portsin.ListarSaldosDoLocalInput) ([]*domainestoque.SaldoLocalDetalhe, error) {
	if f.erro != nil {
		return nil, f.erro
	}
	f.inputSaldos = input

	return f.saldos, nil
}

var _ portsin.LocalUseCase = (*localUseCaseFalso)(nil)

func novoLocalDeTeste(t *testing.T) *domainestoque.Local {
	t.Helper()

	local, err := domainestoque.NovoLocal(uuid.New(), "Galpão Cabedelo")
	if err != nil {
		t.Fatalf("local de teste inválido: %v", err)
	}

	return local
}

func TestCriarLocalRespondeCreatedComOLocal(t *testing.T) {
	local := novoLocalDeTeste(t)
	usecase := &localUseCaseFalso{local: local}

	resposta := executaProtegido(t, NewLocalHandler(usecase).Criar, http.MethodPost, "/api/v1/estoque/locais", "",
		`{"empresa_id":"`+local.EmpresaID.String()+`","nome":"Galpão Cabedelo"}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusCreated, resposta.Body.String())
	}
	if usecase.inputCriar.EmpresaID != local.EmpresaID || usecase.inputCriar.Nome != "Galpão Cabedelo" || usecase.inputCriar.UsuarioID == uuid.Nil {
		t.Errorf("input repassado = %+v", usecase.inputCriar)
	}

	var envelope dto.Resposta[estoquedto.LocalResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	if envelope.Dados.ID != local.ID || envelope.Dados.EmpresaID != local.EmpresaID || !envelope.Dados.Ativo {
		t.Errorf("local devolvido = %+v", envelope.Dados)
	}
}

func TestCriarLocalComCampoDesconhecidoResponde400(t *testing.T) {
	usecase := &localUseCaseFalso{local: novoLocalDeTeste(t)}

	resposta := executaProtegido(t, NewLocalHandler(usecase).Criar, http.MethodPost, "/api/v1/estoque/locais", "",
		`{"nome":"Galpão","saldo":10}`)

	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
	if usecase.inputCriar.Nome != "" {
		t.Error("não deveria chegar ao usecase com corpo inválido")
	}
}

func TestListarLocaisRepassaFiltros(t *testing.T) {
	usecase := &localUseCaseFalso{local: novoLocalDeTeste(t)}
	empresa := uuid.New()

	resposta := executaProtegido(t, NewLocalHandler(usecase).Listar, http.MethodGet,
		"/api/v1/estoque/locais?empresa="+empresa.String()+"&ativos=false&pagina=2&tamanho=5", "", "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	filtro := usecase.inputListar
	if filtro.Empresa == nil || *filtro.Empresa != empresa || filtro.Ativo == nil || *filtro.Ativo || filtro.Filtro.Page != 2 || filtro.Filtro.Size != 5 {
		t.Errorf("filtro repassado = %+v", filtro)
	}

	resposta = executaProtegido(t, NewLocalHandler(usecase).Listar, http.MethodGet, "/api/v1/estoque/locais?empresa=xyz", "", "")
	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
}

func TestAlterarLocalRepassaIdEUsuario(t *testing.T) {
	local := novoLocalDeTeste(t)
	usecase := &localUseCaseFalso{local: local}
	handler := NewLocalHandler(usecase)
	caminho := "/api/v1/estoque/locais/" + local.ID.String()

	resposta := executaProtegido(t, handler.Renomear, http.MethodPut, caminho, local.ID.String(), `{"nome":"Galpão Norte"}`)
	if resposta.Code != http.StatusOK || usecase.inputRenomear.LocalID != local.ID || usecase.inputRenomear.Nome != "Galpão Norte" {
		t.Errorf("renomear: status %d, input %+v", resposta.Code, usecase.inputRenomear)
	}

	resposta = executaProtegido(t, handler.Desativar, http.MethodPatch, caminho+"/desativar", local.ID.String(), "")
	if resposta.Code != http.StatusOK || usecase.desativado != local.ID {
		t.Errorf("desativar: status %d, local %s", resposta.Code, usecase.desativado)
	}

	resposta = executaProtegido(t, handler.Remover, http.MethodDelete, caminho, local.ID.String(), "")
	if resposta.Code != http.StatusNoContent || usecase.removido != local.ID {
		t.Errorf("remover: status %d, local %s", resposta.Code, usecase.removido)
	}
}

func TestErrosDoLocalViramStatusDoEnvelope(t *testing.T) {
	local := novoLocalDeTeste(t)
	casos := []struct {
		nome   string
		erro   error
		status int
	}{
		{"desativar com saldo", domain.ErroValidacao("local com saldo não pode ser desativado"), http.StatusBadRequest},
		{"fora do acesso", domain.ErroPermissao("perfil sem acesso a esta empresa"), http.StatusForbidden},
		{"inexistente", domain.ErroNaoEncontrado("local de estoque não encontrado"), http.StatusNotFound},
		{"nome repetido", domain.ErroConflito("já existe um local com este nome na empresa"), http.StatusConflict},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			usecase := &localUseCaseFalso{local: local, erro: caso.erro}
			resposta := executaProtegido(t, NewLocalHandler(usecase).Desativar, http.MethodPatch,
				"/api/v1/estoque/locais/"+local.ID.String()+"/desativar", local.ID.String(), "")

			apoioteste.VerificarEnvelopeDeErro(t, resposta, caso.status)
		})
	}
}

func TestListarSaldosDoLocalRepassaComSaldo(t *testing.T) {
	local := novoLocalDeTeste(t)
	usecase := &localUseCaseFalso{local: local, saldos: []*domainestoque.SaldoLocalDetalhe{{
		SaldoLocal:    domainestoque.SaldoLocal{ProdutoID: uuid.New(), LocalID: local.ID, Quantidade: 8, Reservado: 3, AtualizadoEm: time.Now().UTC()},
		LocalNome:     local.Nome,
		EmpresaID:     local.EmpresaID,
		ProdutoNome:   "Cimento CP II",
		UnidadeMedida: domainestoque.UnidadeMedidaSc,
	}}}

	resposta := executaProtegido(t, NewLocalHandler(usecase).ListarSaldos, http.MethodGet,
		"/api/v1/estoque/locais/"+local.ID.String()+"/saldos?com_saldo=true", local.ID.String(), "")

	if resposta.Code != http.StatusOK {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusOK, resposta.Body.String())
	}
	if usecase.inputSaldos.LocalID != local.ID || !usecase.inputSaldos.ComSaldo {
		t.Errorf("input repassado = %+v", usecase.inputSaldos)
	}

	var pagina dto.Paginado[estoquedto.SaldoLocalResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &pagina); err != nil {
		t.Fatalf("corpo não é uma página válida: %v", err)
	}
	if len(pagina.Dados) != 1 || pagina.Dados[0].Disponivel != 5 || pagina.Dados[0].UnidadeMedida != "sc" || pagina.Dados[0].ProdutoNome != "Cimento CP II" {
		t.Errorf("saldos devolvidos = %+v", pagina.Dados)
	}
}
