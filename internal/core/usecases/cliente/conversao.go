package cliente

import (
	"context"
	"errors"
	"strings"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	portsin "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/in/cliente"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/cliente"
)

func (u *ClienteUsecaseImpl) ConverterLead(ctx context.Context, input portsin.ConverterLeadInput) (*domaincliente.Cliente, error) {
	perfil, err := u.exigeComercial(ctx, input.UsuarioID)
	if err != nil {
		return nil, err
	}

	lead, err := u.leads.Obter(ctx, input.LeadID)
	if err != nil {
		return nil, err
	}
	if lead == nil {
		return nil, domain.ErroNaoEncontrado("lead não encontrado")
	}

	convertidos, err := u.clientes.Listar(ctx, portsout.ClienteFiltro{
		PaginacaoFiltro: domain.PaginacaoFiltro{Page: 1, Size: 1},
		LeadID:          &lead.ID,
	})
	if err != nil {
		return nil, err
	}
	if len(convertidos) > 0 {
		return nil, erroLeadConvertido()
	}

	dados := input.Dados
	if strings.TrimSpace(dados.Nome) == "" {
		dados.Nome = lead.Nome
	}
	if vazio(dados.EmailNF) && strings.TrimSpace(lead.Email) != "" {
		dados.EmailNF = &lead.Email
	}
	if vazio(dados.Telefone) && strings.TrimSpace(lead.Telefone) != "" {
		dados.Telefone = &lead.Telefone
	}

	documento, err := domaincliente.NovoDocumento(input.Documento)
	if err != nil {
		return nil, err
	}

	validados, err := u.dadosValidados(ctx, perfil, nil, dados)
	if err != nil {
		return nil, err
	}

	existente, err := u.clientes.ObterPorDocumento(ctx, documento.Valor())
	if err != nil {
		return nil, err
	}
	if existente != nil {
		return nil, erroDocumentoRepetido()
	}

	cliente, err := domaincliente.NovoCliente(documento, validados, &lead.ID)
	if err != nil {
		return nil, err
	}

	if _, err := u.clientes.Criar(ctx, cliente); err != nil {
		if errors.Is(err, domain.ErrConflito) {
			return nil, u.conflitoDaConversao(ctx, documento)
		}
		return nil, err
	}

	return cliente, nil
}

func (u *ClienteUsecaseImpl) conflitoDaConversao(ctx context.Context, documento domaincliente.Documento) error {
	existente, err := u.clientes.ObterPorDocumento(ctx, documento.Valor())
	if err != nil {
		return err
	}
	if existente != nil {
		return erroDocumentoRepetido()
	}

	return erroLeadConvertido()
}

func vazio(valor *string) bool {
	return valor == nil || strings.TrimSpace(*valor) == ""
}

func erroLeadConvertido() error {
	return domain.ErroConflito("lead já convertido em cliente")
}
