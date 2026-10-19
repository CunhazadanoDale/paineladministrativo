package pedido

import (
	"strconv"
	"strings"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	"github.com/google/uuid"
)

const pontosBaseDoTodo = 10000

type EmpresaDoDocumento struct {
	ID           uuid.UUID
	RazaoSocial  string
	NomeFantasia string
	CNPJ         string
	UF           string
}

type ClienteDoDocumento struct {
	ID        uuid.UUID
	Nome      string
	Documento string
	Cidade    string
	UF        string
	Telefone  *string
	Email     *string
}

type ObraDoDocumento struct {
	ID       uuid.UUID
	Nome     string
	Endereco string
}

type RepresentanteDoDocumento struct {
	ID       uuid.UUID
	Nome     string
	Email    *string
	Telefone *string
}

type ProdutoDoDocumento struct {
	ID      uuid.UUID
	Codigo  *string
	Nome    string
	Unidade string
}

type PartesDoDocumento struct {
	Empresa       EmpresaDoDocumento
	Cliente       ClienteDoDocumento
	Obra          *ObraDoDocumento
	Representante *RepresentanteDoDocumento
	Produtos      map[uuid.UUID]ProdutoDoDocumento
}

type ItemDoDocumento struct {
	ItemID                   uuid.UUID
	ProdutoID                uuid.UUID
	Codigo                   *string
	Nome                     string
	Unidade                  string
	Quantidade               int
	PrecoUnitarioCentavos    int64
	PrecoTabelaCentavos      *int64
	DescontoUnitarioCentavos *int64
	SubtotalCentavos         int64
}

type DocumentoProposta struct {
	PedidoID            uuid.UUID
	Numero              int
	Revisao             int
	Status              Status
	Modelo              Modelo
	EmitidoEm           time.Time
	Validade            *time.Time
	Vencida             bool
	Empresa             EmpresaDoDocumento
	Cliente             ClienteDoDocumento
	Obra                *ObraDoDocumento
	Representante       *RepresentanteDoDocumento
	Itens               []ItemDoDocumento
	TotalCentavos       int64
	DescontoCentavos    int64
	Condicoes           []string
	Frete               *Frete
	NumeroPedidoCliente *string
	Observacoes         string
}

type ModeloDeVenda struct {
	Modelo    Modelo
	Condicao  Condicao
	Condicoes []string
}

func ModelosDeVenda() []ModeloDeVenda {
	modelos := []Modelo{ModeloProntaEntrega, ModeloImportacao}
	resultado := make([]ModeloDeVenda, 0, len(modelos))
	for _, modelo := range modelos {
		condicao := CondicaoDoModelo(modelo)
		resultado = append(resultado, ModeloDeVenda{
			Modelo:    modelo,
			Condicao:  condicao,
			Condicoes: condicoesPorExtenso(modelo, condicao, nil, nil, nil),
		})
	}

	return resultado
}

func NovoDocumentoProposta(p *Pedido, partes PartesDoDocumento, agora time.Time) (*DocumentoProposta, error) {
	if p == nil {
		return nil, domain.ErroValidacao("documento da proposta sem pedido")
	}

	documento := &DocumentoProposta{
		PedidoID:            p.ID,
		Numero:              p.Numero,
		Revisao:             p.Revisao,
		Status:              p.Status,
		Modelo:              p.Modelo,
		EmitidoEm:           Hoje(agora),
		Validade:            p.Validade,
		Vencida:             p.Vencida(agora),
		Empresa:             partes.Empresa,
		Cliente:             partes.Cliente,
		Obra:                partes.Obra,
		Representante:       partes.Representante,
		Itens:               make([]ItemDoDocumento, 0, len(p.Itens)),
		TotalCentavos:       p.ValorTotalCentavos,
		Condicoes:           p.CondicoesPorExtenso(),
		Frete:               p.Frete,
		NumeroPedidoCliente: p.NumeroPedidoCliente,
		Observacoes:         p.Observacoes,
	}

	for _, item := range p.Itens {
		subtotal, err := item.Subtotal()
		if err != nil {
			return nil, err
		}
		produto := partes.Produtos[item.ProdutoID]
		linha := ItemDoDocumento{
			ItemID:                   item.ID,
			ProdutoID:                item.ProdutoID,
			Codigo:                   produto.Codigo,
			Nome:                     produto.Nome,
			Unidade:                  produto.Unidade,
			Quantidade:               item.Quantidade,
			PrecoUnitarioCentavos:    item.PrecoUnitario.Centavos(),
			DescontoUnitarioCentavos: item.DescontoUnitarioCentavos(),
			SubtotalCentavos:         subtotal,
		}
		if item.PrecoTabela != nil {
			tabela := item.PrecoTabela.Centavos()
			linha.PrecoTabelaCentavos = &tabela
			if tabela > item.PrecoUnitario.Centavos() {
				desconto, err := PrecoDe(tabela - item.PrecoUnitario.Centavos()).Vezes(item.Quantidade)
				if err != nil {
					return nil, err
				}
				if documento.DescontoCentavos, err = somarCentavos(documento.DescontoCentavos, desconto); err != nil {
					return nil, err
				}
			}
		}
		documento.Itens = append(documento.Itens, linha)
	}

	return documento, nil
}

func (p *Pedido) CondicoesPorExtenso() []string {
	return condicoesPorExtenso(p.Modelo, p.Condicao, p.PrazoLimite, p.Frete, p.validadeDaProposta())
}

func (p *Pedido) validadeDaProposta() *time.Time {
	if p.Status != StatusProposta {
		return nil
	}

	return p.Validade
}

func condicoesPorExtenso(modelo Modelo, condicao Condicao, prazoLimite *time.Time, frete *Frete, validade *time.Time) []string {
	entrada := condicao.Entrada().PontosBase()
	saldo := pontosBaseDoTodo - entrada
	condicoes := make([]string, 0, 5)

	if entrada > 0 {
		condicoes = append(condicoes, "Entrada de "+formatarPercentual(entrada)+" na confirmação do pedido.")
	}

	momentoDoSaldo := "na entrega"
	if modelo == ModeloImportacao {
		momentoDoSaldo = "na chegada da mercadoria"
	}
	if saldo > 0 {
		prefixo := "Pagamento"
		if entrada > 0 {
			prefixo = "Saldo de " + formatarPercentual(saldo)
		}
		if prazo := condicao.PrazoBoletoDias(); prazo != nil {
			apos := "após a entrega"
			if modelo == ModeloImportacao {
				apos = "após a chegada da mercadoria"
			}
			condicoes = append(condicoes, prefixo+" em boleto com vencimento em "+strconv.Itoa(*prazo)+" dias "+apos+".")
		} else {
			condicoes = append(condicoes, prefixo+" "+momentoDoSaldo+".")
		}
	}

	if prazo := condicao.PrazoEntregaDias(); prazo != nil && modelo == ModeloImportacao {
		contados := "contados do pagamento da entrada"
		if entrada == 0 {
			contados = "contados da confirmação do pedido"
		}
		condicoes = append(condicoes, "Prazo de entrega de "+strconv.Itoa(*prazo)+" dias "+contados+".")
	}
	if prazoLimite != nil {
		condicoes = append(condicoes, "Entrega até "+prazoLimite.Format("02/01/2006")+".")
	}

	if frete != nil {
		texto := "Frete FOB (por conta do cliente)"
		if frete.Tipo() == TipoFreteCIF {
			texto = "Frete CIF (por conta do vendedor)"
		}
		if valor := frete.Centavos(); valor != nil {
			texto += ", estimado em " + formatarReais(*valor)
		}
		condicoes = append(condicoes, texto+".")
	}
	if validade != nil {
		condicoes = append(condicoes, "Proposta válida até "+validade.Format("02/01/2006")+".")
	}

	return condicoes
}

func formatarPercentual(pontosBase int) string {
	inteiro, fracao := pontosBase/100, pontosBase%100
	if fracao == 0 {
		return strconv.Itoa(inteiro) + "%"
	}

	decimais := strings.TrimRight(strconv.Itoa(fracao + 100)[1:], "0")

	return strconv.Itoa(inteiro) + "," + decimais + "%"
}

func formatarReais(centavos int64) string {
	sinal := ""
	if centavos < 0 {
		sinal = "-"
		centavos = -centavos
	}

	reais := strconv.FormatInt(centavos/100, 10)
	grupos := make([]string, 0, len(reais)/3+1)
	for len(reais) > 3 {
		grupos = append([]string{reais[len(reais)-3:]}, grupos...)
		reais = reais[:len(reais)-3]
	}
	grupos = append([]string{reais}, grupos...)

	return sinal + "R$ " + strings.Join(grupos, ".") + "," + strconv.FormatInt(centavos%100+100, 10)[1:]
}
