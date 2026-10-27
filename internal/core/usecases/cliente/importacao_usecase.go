package cliente

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	"github.com/google/uuid"
)

var _ portsin.ImportacaoUseCase = (*ClienteUsecaseImpl)(nil)

type importacao struct {
	simular        bool
	grupos         map[string]*uuid.UUID
	representantes map[string]*uuid.UUID
	documentos     map[string]int
}

func (u *ClienteUsecaseImpl) Importar(ctx context.Context, input portsin.ImportarClientesInput) (*domaincliente.RelatorioImportacao, error) {
	perfil, err := u.perfil(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}
	if !perfil.administrador {
		return nil, domain.ErroPermissao("perfil sem permissão para esta operação")
	}
	if len(input.Linhas) > domaincliente.LimiteDeLinhasDaImportacao {
		return nil, domain.ErroValidacao("arquivo com mais de " + strconv.Itoa(domaincliente.LimiteDeLinhasDaImportacao) + " linhas")
	}

	relatorio := &domaincliente.RelatorioImportacao{
		Simulacao:  input.Simular,
		Total:      len(input.Linhas),
		Rejeitadas: []domaincliente.LinhaRejeitada{},
	}
	estado := &importacao{
		simular:        input.Simular,
		grupos:         map[string]*uuid.UUID{},
		representantes: map[string]*uuid.UUID{},
		documentos:     map[string]int{},
	}

	for _, linha := range input.Linhas {
		motivo, err := u.importarLinha(ctx, perfil, estado, linha)
		if err != nil {
			return nil, err
		}
		if motivo != "" {
			relatorio.Rejeitar(linha.Numero, strings.TrimSpace(linha.Documento), motivo)
			continue
		}
		relatorio.Importadas++
	}

	return relatorio, nil
}

func (u *ClienteUsecaseImpl) importarLinha(ctx context.Context, perfil perfilDoUsuario, estado *importacao, linha portsin.LinhaImportacao) (string, error) {
	documento, err := domaincliente.NovoDocumento(linha.Documento)
	if err != nil {
		return motivoDe(err)
	}
	if anterior, repetido := estado.documentos[documento.Valor()]; repetido {
		return "documento repetido no arquivo (linha " + strconv.Itoa(anterior) + ")", nil
	}

	existente, err := u.clientes.ObterPorDocumento(ctx, documento.Valor())
	if err != nil {
		return "", err
	}
	if existente != nil {
		return "documento já cadastrado", nil
	}

	dados, err := u.dadosDaLinha(ctx, estado, linha)
	if err != nil {
		return motivoDe(err)
	}

	validados, err := u.dadosValidados(ctx, perfil, nil, dados)
	if err != nil {
		return motivoDe(err)
	}

	cliente, err := domaincliente.NovoCliente(documento, validados, nil)
	if err != nil {
		return motivoDe(err)
	}

	if cliente.GrupoID, err = u.grupoDaLinha(ctx, estado, linha.Grupo); err != nil {
		return motivoDe(err)
	}

	if !estado.simular {
		if _, err := u.clientes.Criar(ctx, cliente); err != nil {
			if errors.Is(err, domain.ErrConflito) {
				return "documento já cadastrado", nil
			}
			return motivoDe(err)
		}
	}

	estado.documentos[documento.Valor()] = linha.Numero

	return "", nil
}

func (u *ClienteUsecaseImpl) dadosDaLinha(ctx context.Context, estado *importacao, linha portsin.LinhaImportacao) (portsin.DadosClienteInput, error) {
	dados := portsin.DadosClienteInput{
		Nome:         linha.Nome,
		NomeFantasia: &linha.NomeFantasia,
		Tipo:         linha.Tipo,
		UF:           linha.UF,
		Cidade:       linha.Cidade,
		EmailNF:      &linha.EmailNF,
		Telefone:     &linha.Telefone,
		Observacoes:  linha.Observacoes,
	}

	var err error
	if dados.ExigeNumeroPedidoNaNF, err = domain.FlagDeImportacao(linha.ExigeNumeroPedidoNaNF, "exige_numero_pedido_na_nf"); err != nil {
		return dados, err
	}
	if dados.ProibeAntecipacao, err = domain.FlagDeImportacao(linha.ProibeAntecipacao, "proibe_antecipacao"); err != nil {
		return dados, err
	}
	if dados.RepresentanteID, err = u.representanteDaLinha(ctx, estado, linha.RepresentanteDocumento); err != nil {
		return dados, err
	}

	return dados, nil
}

func (u *ClienteUsecaseImpl) grupoDaLinha(ctx context.Context, estado *importacao, nome string) (*uuid.UUID, error) {
	chave := domaincliente.NomeDeBusca(nome)
	if chave == "" {
		return nil, nil
	}
	if id, ok := estado.grupos[chave]; ok {
		return id, nil
	}

	existente, err := u.grupos.ObterPorNomeBusca(ctx, chave)
	if err != nil {
		return nil, err
	}
	if existente != nil {
		if !existente.Ativo {
			return nil, domain.ErroValidacao("grupo de clientes inativo não recebe cliente")
		}
		estado.grupos[chave] = &existente.ID
		return &existente.ID, nil
	}

	grupo, err := domaincliente.NovoGrupo(nome)
	if err != nil {
		return nil, err
	}
	if estado.simular {
		estado.grupos[chave] = nil
		return nil, nil
	}
	if _, err := u.grupos.Criar(ctx, grupo); err != nil {
		return nil, traduzirGrupoRepetido(err)
	}
	estado.grupos[chave] = &grupo.ID

	return &grupo.ID, nil
}

func (u *ClienteUsecaseImpl) representanteDaLinha(ctx context.Context, estado *importacao, valor string) (*uuid.UUID, error) {
	if strings.TrimSpace(valor) == "" {
		return nil, nil
	}

	documento, err := domaincliente.NovoDocumento(valor)
	if err != nil {
		return nil, domain.ErroValidacao("documento do representante inválido")
	}
	if id, ok := estado.representantes[documento.Valor()]; ok {
		return id, nil
	}

	representante, err := u.representantes.ObterPorDocumento(ctx, documento.Valor())
	if err != nil {
		return nil, err
	}
	if representante == nil {
		return nil, domain.ErroValidacao("representante não encontrado pelo documento")
	}
	estado.representantes[documento.Valor()] = &representante.ID

	return &representante.ID, nil
}

func motivoDe(err error) (string, error) {
	for _, sentinela := range []error{domain.ErrValidacao, domain.ErrConflito, domain.ErrNotFound} {
		if errors.Is(err, sentinela) {
			return strings.TrimPrefix(err.Error(), sentinela.Error()+": "), nil
		}
	}

	return "", err
}
