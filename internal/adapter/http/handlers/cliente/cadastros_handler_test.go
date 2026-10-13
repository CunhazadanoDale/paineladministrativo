package cliente

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/apoioteste"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto"
	clientedto "github.com/CunhazadanoDale/paineladministrativo.git/internal/adapter/http/dto/cliente"
	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	"github.com/google/uuid"
)

type representanteUseCaseFalso struct {
	portsin.RepresentanteUseCase
	erro          error
	representante *domaincliente.Representante
	inputCriar    portsin.CriarRepresentanteInput
}

func (f *representanteUseCaseFalso) Criar(_ context.Context, input portsin.CriarRepresentanteInput) (*domaincliente.Representante, error) {
	f.inputCriar = input
	return f.representante, f.erro
}

type obraUseCaseFalso struct {
	portsin.ObraUseCase
	obra        *domaincliente.Obra
	inputCriar  portsin.CriarObraInput
	inputListar portsin.ListarObrasInput
}

func (f *obraUseCaseFalso) Criar(_ context.Context, input portsin.CriarObraInput) (*domaincliente.Obra, error) {
	f.inputCriar = input
	return f.obra, nil
}

func (f *obraUseCaseFalso) Listar(_ context.Context, input portsin.ListarObrasInput) ([]*domaincliente.Obra, error) {
	f.inputListar = input
	return []*domaincliente.Obra{f.obra}, nil
}

type grupoUseCaseFalso struct {
	portsin.GrupoUseCase
	grupo       *domaincliente.Grupo
	inputListar portsin.ListarGruposInput
	desativado  uuid.UUID
}

func (f *grupoUseCaseFalso) Listar(_ context.Context, input portsin.ListarGruposInput) ([]*domaincliente.Grupo, error) {
	f.inputListar = input
	return []*domaincliente.Grupo{f.grupo}, nil
}

func (f *grupoUseCaseFalso) Desativar(_ context.Context, grupoID, _ uuid.UUID) (*domaincliente.Grupo, error) {
	f.desativado = grupoID
	return f.grupo, nil
}

func TestCriarRepresentanteRepassaVinculoEPercentuais(t *testing.T) {
	documento, _ := domaincliente.NovoDocumento("529.982.247-25")
	usuarioID := uuid.New()
	representante, err := domaincliente.NovoRepresentante(domaincliente.DadosRepresentante{
		Nome: "José Vendas", Documento: &documento, UsuarioID: &usuarioID, ComissaoProntaEntrega: domaincliente.PercentualDe(300),
	})
	if err != nil {
		t.Fatalf("representante de teste inválido: %v", err)
	}
	usecase := &representanteUseCaseFalso{representante: representante}

	resposta := executaProtegido(t, NewRepresentanteHandler(usecase).Criar, http.MethodPost, "/api/v1/representantes", "",
		`{"nome":"José Vendas","documento":"529.982.247-25","usuario_id":"`+usuarioID.String()+`","comissao_pronta_entrega_bp":300,"comissao_importacao_bp":150}`)

	if resposta.Code != http.StatusCreated {
		t.Fatalf("status %d, esperado %d: %s", resposta.Code, http.StatusCreated, resposta.Body.String())
	}
	dados := usecase.inputCriar.Dados
	if *dados.UsuarioVinculadoID != usuarioID || dados.ComissaoProntaEntregaBP != 300 || dados.ComissaoImportacaoBP != 150 || *dados.Documento != "529.982.247-25" {
		t.Errorf("dados repassados = %+v", dados)
	}

	var envelope dto.Resposta[clientedto.RepresentanteResponse]
	if err := json.Unmarshal(resposta.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("corpo não é um envelope válido: %v", err)
	}
	if *envelope.Dados.DocumentoFormatado != "529.982.247-25" || envelope.Dados.ComissaoProntaEntregaBP != 300 {
		t.Errorf("representante devolvido = %+v", envelope.Dados)
	}

	usecase.erro = domain.ErroPermissao("perfil sem permissão para esta operação")
	resposta = executaProtegido(t, NewRepresentanteHandler(usecase).Criar, http.MethodPost, "/api/v1/representantes", "", `{"nome":"X"}`)
	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusForbidden)
}

func TestObrasDoClienteUsamOIdDaRota(t *testing.T) {
	clienteID := uuid.New()
	obra, err := domaincliente.NovaObra(clienteID, domaincliente.DadosObra{
		Nome:     "Residencial Mar Azul",
		Endereco: domaincliente.Endereco{Logradouro: "Av. Beira Mar", Cidade: "Cabedelo", UF: domainempresa.UFDe("PB")},
	})
	if err != nil {
		t.Fatalf("obra de teste inválida: %v", err)
	}
	usecase := &obraUseCaseFalso{obra: obra}
	caminho := "/api/v1/clientes/" + clienteID.String() + "/obras"

	resposta := executaProtegido(t, NewObraHandler(usecase).Criar, http.MethodPost, caminho, clienteID.String(),
		`{"nome":"Residencial Mar Azul","logradouro":"Av. Beira Mar","cidade":"Cabedelo","uf":"PB","cep":"58310-000"}`)
	if resposta.Code != http.StatusCreated || usecase.inputCriar.ClienteID != clienteID || *usecase.inputCriar.Dados.CEP != "58310-000" {
		t.Errorf("criar obra: status %d, input %+v", resposta.Code, usecase.inputCriar)
	}

	resposta = executaProtegido(t, NewObraHandler(usecase).Listar, http.MethodGet, caminho+"?ativas=false", clienteID.String(), "")
	if resposta.Code != http.StatusOK || usecase.inputListar.ClienteID != clienteID || usecase.inputListar.Ativo == nil || *usecase.inputListar.Ativo {
		t.Errorf("listar obras: status %d, input %+v", resposta.Code, usecase.inputListar)
	}
}

func TestGruposRepassamFiltrosEIdDaRota(t *testing.T) {
	grupo, _ := domaincliente.NovoGrupo("Grupo Ômega")
	usecase := &grupoUseCaseFalso{grupo: grupo}

	resposta := executaProtegido(t, NewGrupoHandler(usecase).Listar, http.MethodGet, "/api/v1/cliente-grupos?busca=omega&ativos=true", "", "")
	if resposta.Code != http.StatusOK || usecase.inputListar.Busca != "omega" || usecase.inputListar.Ativo == nil || !*usecase.inputListar.Ativo {
		t.Errorf("listar grupos: status %d, input %+v", resposta.Code, usecase.inputListar)
	}

	resposta = executaProtegido(t, NewGrupoHandler(usecase).Desativar, http.MethodPatch,
		"/api/v1/cliente-grupos/"+grupo.ID.String()+"/desativar", grupo.ID.String(), "")
	if resposta.Code != http.StatusOK || usecase.desativado != grupo.ID {
		t.Errorf("desativar grupo: status %d, grupo %s", resposta.Code, usecase.desativado)
	}

	resposta = executaProtegido(t, NewGrupoHandler(usecase).Obter, http.MethodGet, "/api/v1/cliente-grupos/abc", "abc", "")
	apoioteste.VerificarEnvelopeDeErro(t, resposta, http.StatusBadRequest)
}
