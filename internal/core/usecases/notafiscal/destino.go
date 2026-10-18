package notafiscal

import (
	"context"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainempresa "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/empresa"
	domainnotafiscal "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/notafiscal"
)

const tamanhoCNPJ = 14

type destinoDaNota struct {
	empresa     *domainempresa.Empresa
	cliente     *domaincliente.Cliente
	clienteNovo bool
}

func (u *NotaFiscalUsecaseImpl) destinoDaNota(ctx context.Context, documento *domainnotafiscal.Documento) (*destinoDaNota, error) {
	destinatario := documento.Destinatario

	if len(destinatario.Documento) == tamanhoCNPJ {
		empresa, err := u.empresas.ObterPorCNPJ(ctx, domainempresa.CNPJDe(destinatario.Documento))
		if err != nil {
			return nil, err
		}
		if empresa != nil {
			return &destinoDaNota{empresa: empresa}, nil
		}
	}

	cliente, err := u.clientes.ObterPorDocumento(ctx, destinatario.Documento)
	if err != nil {
		return nil, err
	}
	if cliente != nil {
		return &destinoDaNota{cliente: cliente}, nil
	}

	novo, err := clienteDoDestinatario(destinatario)
	if err != nil {
		return nil, err
	}

	return &destinoDaNota{cliente: novo, clienteNovo: true}, nil
}

func clienteDoDestinatario(destinatario domainnotafiscal.Destinatario) (*domaincliente.Cliente, error) {
	documento, err := domaincliente.NovoDocumento(destinatario.Documento)
	if err != nil {
		return nil, err
	}

	uf, err := domainempresa.NovaUF(destinatario.UF)
	if err != nil {
		return nil, domain.ErroValidacao("cliente novo sem UF válida no endereço do destinatário")
	}
	if destinatario.Cidade == "" {
		return nil, domain.ErroValidacao("cliente novo sem cidade no endereço do destinatário")
	}

	tipo := domaincliente.TipoClienteOutro
	if documento.Pessoa() == domaincliente.PessoaFisica {
		tipo = domaincliente.TipoClientePF
	}

	dados := domaincliente.DadosCliente{
		Nome:   destinatario.Nome,
		Tipo:   tipo,
		UF:     uf,
		Cidade: destinatario.Cidade,
	}
	if destinatario.Email != "" {
		email := destinatario.Email
		dados.EmailNF = &email
	}

	cliente, err := domaincliente.NovoCliente(documento, dados, nil)
	if err != nil && dados.EmailNF != nil {
		dados.EmailNF = nil
		cliente, err = domaincliente.NovoCliente(documento, dados, nil)
	}
	if err != nil {
		return nil, err
	}

	return cliente, nil
}
