package pedido

import (
	"context"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domaincliente "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/cliente"
	domainpedido "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/pedido"
	"github.com/google/uuid"
)

func (u *PedidoUsecaseImpl) Documento(ctx context.Context, pedidoID, usuarioID uuid.UUID) (*domainpedido.DocumentoProposta, error) {
	perfil, err := u.perfil(ctx, usuarioID)
	if err != nil {
		return nil, err
	}

	pedido, err := u.pedidoVisivel(ctx, perfil, pedidoID)
	if err != nil {
		return nil, err
	}

	partes, err := u.partesDoDocumento(ctx, pedido)
	if err != nil {
		return nil, err
	}

	return domainpedido.NovoDocumentoProposta(pedido, partes, time.Now().UTC())
}

func (u *PedidoUsecaseImpl) ModelosDeVenda(context.Context) []domainpedido.ModeloDeVenda {
	return domainpedido.ModelosDeVenda()
}

func (u *PedidoUsecaseImpl) partesDoDocumento(ctx context.Context, pedido *domainpedido.Pedido) (domainpedido.PartesDoDocumento, error) {
	empresa, err := u.empresas.Obter(ctx, pedido.EmpresaID)
	if err != nil {
		return domainpedido.PartesDoDocumento{}, err
	}
	if empresa == nil {
		return domainpedido.PartesDoDocumento{}, domain.ErroValidacao("empresa do pedido não encontrada")
	}

	cliente, err := u.clientes.Obter(ctx, pedido.ClienteID)
	if err != nil {
		return domainpedido.PartesDoDocumento{}, err
	}
	if cliente == nil {
		return domainpedido.PartesDoDocumento{}, domain.ErroValidacao("cliente do pedido não encontrado")
	}

	partes := domainpedido.PartesDoDocumento{
		Empresa: domainpedido.EmpresaDoDocumento{
			ID:           empresa.ID,
			RazaoSocial:  empresa.RazaoSocial,
			NomeFantasia: empresa.NomeFantasia,
			CNPJ:         empresa.CNPJ.Formatado(),
			UF:           empresa.UF.String(),
		},
		Cliente: domainpedido.ClienteDoDocumento{
			ID:        cliente.ID,
			Nome:      cliente.Nome,
			Documento: cliente.Documento.Formatado(),
			Cidade:    cliente.Cidade,
			UF:        cliente.UF.String(),
			Telefone:  cliente.Telefone,
			Email:     cliente.EmailNF,
		},
		Produtos: make(map[uuid.UUID]domainpedido.ProdutoDoDocumento, len(pedido.Itens)),
	}

	if pedido.ObraID != nil {
		obra, err := u.obras.Obter(ctx, *pedido.ObraID)
		if err != nil {
			return domainpedido.PartesDoDocumento{}, err
		}
		if obra != nil {
			partes.Obra = &domainpedido.ObraDoDocumento{ID: obra.ID, Nome: obra.Nome, Endereco: enderecoPorExtenso(obra.Endereco)}
		}
	}

	if pedido.RepresentanteID != nil {
		representante, err := u.representantes.Obter(ctx, *pedido.RepresentanteID)
		if err != nil {
			return domainpedido.PartesDoDocumento{}, err
		}
		if representante != nil {
			partes.Representante = &domainpedido.RepresentanteDoDocumento{
				ID: representante.ID, Nome: representante.Nome, Email: representante.Email, Telefone: representante.Telefone,
			}
		}
	}

	for _, item := range pedido.Itens {
		if _, carregado := partes.Produtos[item.ProdutoID]; carregado {
			continue
		}
		produto, err := u.produtos.Obter(ctx, item.ProdutoID)
		if err != nil {
			return domainpedido.PartesDoDocumento{}, err
		}
		if produto == nil {
			continue
		}
		partes.Produtos[item.ProdutoID] = domainpedido.ProdutoDoDocumento{
			ID: produto.ID, Codigo: produto.Codigo, Nome: produto.Nome, Unidade: produto.UnidadeMedida.String(),
		}
	}

	return partes, nil
}

func enderecoPorExtenso(endereco domaincliente.Endereco) string {
	partes := []string{endereco.Logradouro}
	if endereco.Numero != nil {
		partes[0] += ", " + *endereco.Numero
	}
	if endereco.Complemento != nil {
		partes = append(partes, *endereco.Complemento)
	}
	if endereco.Bairro != nil {
		partes = append(partes, *endereco.Bairro)
	}
	cidade := endereco.Cidade
	if uf := endereco.UF.String(); uf != "" {
		cidade += "/" + uf
	}
	partes = append(partes, cidade)
	if endereco.CEP != nil {
		partes = append(partes, "CEP "+*endereco.CEP)
	}

	return strings.Join(partes, " — ")
}
