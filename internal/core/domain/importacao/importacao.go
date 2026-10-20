package importacao

import (
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainfornecedor "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/fornecedor"
	"github.com/google/uuid"
)

const (
	versaoInicial                   = 1
	tamanhoMaximoReferencia         = 60
	tamanhoMaximoDocumentoEmbarque  = 60
	tamanhoMaximoDeclaracao         = 30
	tamanhoMaximoObservacoes        = 2000
	tamanhoMaximoMotivoCancelamento = 1000
)

type Importacao struct {
	ID                 uuid.UUID
	EmpresaID          uuid.UUID
	Numero             int
	FornecedorID       uuid.UUID
	Referencia         *string
	Moeda              domainfornecedor.Moeda
	Status             Status
	ValorTotal         ValorMoeda
	EmbarquePrevisto   *time.Time
	EmbarcadaEm        *time.Time
	ChegadaPrevista    *time.Time
	ChegadaEm          *time.Time
	DesembaracadaEm    *time.Time
	DocumentoEmbarque  *string
	Declaracao         *string
	Observacoes        string
	CanceladaMotivo    *string
	UsuarioID          uuid.UUID
	Versao             int
	CriadoEm           time.Time
	AtualizadoEm       time.Time
	Itens              []*Item
	Cambio             *Cambio
	RecebidaEm         *time.Time
	LocalRecebimentoID *uuid.UUID
	Parcelas           []*Parcela
	Custos             []*Custo
	Alocacoes          []*Alocacao
}

type DadosImportacao struct {
	FornecedorID     uuid.UUID
	Moeda            domainfornecedor.Moeda
	Referencia       *string
	EmbarquePrevisto *time.Time
	ChegadaPrevista  *time.Time
	Observacoes      string
	Itens            []ItemInput
}

type EmbarqueInput struct {
	EmbarcadaEm       time.Time
	ChegadaPrevista   time.Time
	DocumentoEmbarque *string
}

type DesembaracoInput struct {
	ChegadaEm       time.Time
	DesembaracadaEm time.Time
	Declaracao      string
}

func NovaImportacao(empresaID, usuarioID uuid.UUID, dados DadosImportacao, agora time.Time) (*Importacao, error) {
	if empresaID == uuid.Nil {
		return nil, domain.ErroValidacao("empresa da importação é obrigatória")
	}
	if usuarioID == uuid.Nil {
		return nil, domain.ErroValidacao("usuário da importação não informado")
	}

	importacao := &Importacao{
		ID:        uuid.Must(uuid.NewV7()),
		EmpresaID: empresaID,
		Status:    StatusAberta,
		UsuarioID: usuarioID,
		Versao:    versaoInicial,
		CriadoEm:  agora,
	}

	if err := importacao.aplicarDados(dados, agora); err != nil {
		return nil, err
	}

	return importacao, nil
}

func ErroVersaoDivergente() error {
	return domain.ErroConflito("a importação foi alterada por outra pessoa; recarregue e tente de novo")
}

func (i *Importacao) ConferirVersao(informada int) error {
	if informada < versaoInicial {
		return domain.ErroValidacao("versão da importação é obrigatória")
	}
	if informada != i.Versao {
		return ErroVersaoDivergente()
	}

	return nil
}

func (i *Importacao) Alterar(dados DadosImportacao, agora time.Time) error {
	if !i.Status.Editavel() {
		return domain.ErroConflito("só a importação aberta ou embarcada pode ser alterada")
	}
	if i.Status != StatusAberta && (dados.FornecedorID != i.FornecedorID || dados.Moeda != i.Moeda) {
		return domain.ErroConflito("fornecedor e moeda só mudam enquanto a importação está aberta")
	}
	if i.Status != StatusAberta && dataOpcional(dados.ChegadaPrevista) == nil {
		return domain.ErroValidacao("chegada prevista é obrigatória depois do embarque")
	}

	return i.aplicarDados(dados, agora)
}

func (i *Importacao) Embarcar(input EmbarqueInput, agora time.Time) error {
	if !i.Status.PodeTransicionarPara(StatusEmbarcada) {
		return erroTransicao(i.Status, StatusEmbarcada)
	}

	embarcadaEm, err := dataDoFato(input.EmbarcadaEm, agora, feminino("data do embarque"))
	if err != nil {
		return err
	}
	if input.ChegadaPrevista.IsZero() {
		return domain.ErroValidacao(feminino("chegada prevista").obrigatorio())
	}
	chegadaPrevista := DataDe(input.ChegadaPrevista)
	if chegadaPrevista.Before(embarcadaEm) {
		return domain.ErroValidacao("chegada prevista não pode ser anterior ao embarque")
	}
	documento, err := textoOpcional(input.DocumentoEmbarque, "documento de embarque", tamanhoMaximoDocumentoEmbarque)
	if err != nil {
		return err
	}

	if i.EmbarquePrevisto != nil && chegadaPrevista.Before(*i.EmbarquePrevisto) {
		i.EmbarquePrevisto = nil
	}
	i.Status = StatusEmbarcada
	i.EmbarcadaEm = &embarcadaEm
	i.ChegadaPrevista = &chegadaPrevista
	i.DocumentoEmbarque = documento
	i.AtualizadoEm = agora

	return nil
}

func (i *Importacao) Desembaracar(input DesembaracoInput, agora time.Time) error {
	if !i.Status.PodeTransicionarPara(StatusDesembaracada) {
		return erroTransicao(i.Status, StatusDesembaracada)
	}

	chegadaEm, err := dataDoFato(input.ChegadaEm, agora, feminino("data da chegada"))
	if err != nil {
		return err
	}
	if chegadaEm.Before(*i.EmbarcadaEm) {
		return domain.ErroValidacao("data da chegada não pode ser anterior ao embarque em " + formatarData(*i.EmbarcadaEm))
	}
	desembaracadaEm, err := dataDoFato(input.DesembaracadaEm, agora, feminino("data do desembaraço"))
	if err != nil {
		return err
	}
	if desembaracadaEm.Before(chegadaEm) {
		return domain.ErroValidacao("data do desembaraço não pode ser anterior à chegada")
	}
	declaracao, err := textoObrigatorio(input.Declaracao, masculino("número da declaração de importação"), tamanhoMaximoDeclaracao)
	if err != nil {
		return err
	}

	i.Status = StatusDesembaracada
	i.ChegadaEm = &chegadaEm
	i.DesembaracadaEm = &desembaracadaEm
	i.Declaracao = &declaracao
	i.AtualizadoEm = agora

	return nil
}

func (i *Importacao) Cancelar(motivo string, agora time.Time) error {
	limpo, err := textoObrigatorio(motivo, masculino("motivo do cancelamento"), tamanhoMaximoMotivoCancelamento)
	if err != nil {
		return err
	}
	if !i.Status.PodeTransicionarPara(StatusCancelada) {
		return erroTransicao(i.Status, StatusCancelada)
	}
	if len(i.ContasAPagarAbertas()) > 0 {
		return ErroContasAPagarAbertas()
	}

	i.Status = StatusCancelada
	i.CanceladaMotivo = &limpo
	i.AtualizadoEm = agora

	return nil
}

func (i *Importacao) Item(id uuid.UUID) *Item {
	for _, item := range i.Itens {
		if item.ID == id {
			return item
		}
	}

	return nil
}

func (i *Importacao) itemDoProduto(produtoID uuid.UUID) *Item {
	for _, item := range i.Itens {
		if item.ProdutoID == produtoID {
			return item
		}
	}

	return nil
}

func (i *Importacao) aplicarDados(dados DadosImportacao, agora time.Time) error {
	if dados.FornecedorID == uuid.Nil {
		return domain.ErroValidacao("fornecedor da importação é obrigatório")
	}
	if err := dados.Moeda.Validado(); err != nil {
		return err
	}
	if len(dados.Itens) == 0 {
		return domain.ErroValidacao("a importação precisa de pelo menos um item")
	}

	referencia, err := textoOpcional(dados.Referencia, "referência do fornecedor", tamanhoMaximoReferencia)
	if err != nil {
		return err
	}
	observacoes, err := textoLimitado(dados.Observacoes, "observações", tamanhoMaximoObservacoes)
	if err != nil {
		return err
	}
	embarquePrevisto := dataOpcional(dados.EmbarquePrevisto)
	chegadaPrevista := dataOpcional(dados.ChegadaPrevista)
	if embarquePrevisto != nil && chegadaPrevista != nil && chegadaPrevista.Before(*embarquePrevisto) {
		return domain.ErroValidacao("chegada prevista não pode ser anterior ao embarque previsto")
	}
	if i.EmbarcadaEm != nil && chegadaPrevista != nil && chegadaPrevista.Before(*i.EmbarcadaEm) {
		return domain.ErroValidacao("chegada prevista não pode ser anterior ao embarque")
	}

	itens := make([]*Item, 0, len(dados.Itens))
	produtos := make(map[uuid.UUID]bool, len(dados.Itens))
	var total ValorMoeda
	for _, input := range dados.Itens {
		if produtos[input.ProdutoID] {
			return domain.ErroValidacao("produto repetido na importação; some as quantidades em um item")
		}
		produtos[input.ProdutoID] = true

		item, err := novoItem(i.ID, i.itemDoProduto(input.ProdutoID), input, agora)
		if err != nil {
			return err
		}
		if total, err = total.Somar(item.Valor); err != nil {
			return err
		}
		itens = append(itens, item)
	}
	if err := i.conferirParcelasNoTotal(total, ValorMoeda{}, uuid.Nil); err != nil {
		return err
	}
	if err := i.conferirAlocacoesNosItens(itens); err != nil {
		return err
	}

	i.FornecedorID = dados.FornecedorID
	i.Moeda = dados.Moeda
	i.Referencia = referencia
	i.EmbarquePrevisto = embarquePrevisto
	i.ChegadaPrevista = chegadaPrevista
	i.Observacoes = observacoes
	i.Itens = itens
	i.ValorTotal = total
	i.AtualizadoEm = agora

	return nil
}

func (i *Importacao) conferirAlocacoesNosItens(itens []*Item) error {
	quantidades := make(map[uuid.UUID]int, len(itens))
	for _, item := range itens {
		quantidades[item.ID] = item.Quantidade
	}

	alocado := make(map[uuid.UUID]int)
	for _, alocacao := range i.Alocacoes {
		alocado[alocacao.ImportacaoItemID] += alocacao.Quantidade
	}
	for itemID, quantidade := range alocado {
		restante, ok := quantidades[itemID]
		if !ok {
			return domain.ErroValidacao("o item removido tem pedidos alocados; tire as alocações antes")
		}
		if quantidade > restante {
			return domain.ErroValidacao("a quantidade do item ficaria menor que o alocado para pedidos")
		}
	}

	return nil
}
