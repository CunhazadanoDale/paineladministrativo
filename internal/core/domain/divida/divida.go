package divida

import (
	"fmt"
	"sort"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	"github.com/google/uuid"
)

const (
	versaoInicial            = 1
	tamanhoMaximoDescricao   = 120
	tamanhoMaximoContrato    = 60
	tamanhoMaximoObservacoes = 2000
	tamanhoMaximoMotivo      = 1000
	valorMaximoCentavos      = 1_000_000_000_000_000
	maximoDeParcelas         = 600
)

type Divida struct {
	ID                     uuid.UUID
	EmpresaID              uuid.UUID
	CredorTipo             TipoCredor
	FornecedorID           uuid.UUID
	Descricao              string
	Contrato               *string
	ValorOriginalCentavos  *int64
	ContratadaEm           *time.Time
	Taxa                   *Taxa
	Indexador              Indexador
	SaldoInicialCentavos   int64
	DataSaldo              time.Time
	ContaLiberacaoID       *uuid.UUID
	LiberadaEm             *time.Time
	CategoriaAmortizacaoID uuid.UUID
	CategoriaJurosID       uuid.UUID
	DividaOrigemID         *uuid.UUID
	Status                 Status
	Motivo                 *string
	Observacoes            string
	UsuarioID              uuid.UUID
	Versao                 int
	CriadoEm               time.Time
	AtualizadoEm           time.Time
	Parcelas               []*Parcela
}

type DadosInformativos struct {
	CredorTipo             TipoCredor
	Descricao              string
	Contrato               *string
	ValorOriginalCentavos  *int64
	ContratadaEm           *time.Time
	Taxa                   *Taxa
	Indexador              Indexador
	ContaLiberacaoID       *uuid.UUID
	LiberadaEm             *time.Time
	CategoriaAmortizacaoID uuid.UUID
	CategoriaJurosID       uuid.UUID
	Observacoes            string
}

type DadosDivida struct {
	DadosInformativos
	FornecedorID         uuid.UUID
	SaldoInicialCentavos int64
	DataSaldo            time.Time
	Parcelas             []ParcelaInput
}

type ContaCancelada struct {
	SolicitacaoID  uuid.UUID
	StatusAnterior domainsolicitacao.Status
}

func NovaDivida(empresaID, usuarioID uuid.UUID, dados DadosDivida, agora time.Time) (*Divida, error) {
	switch {
	case empresaID == uuid.Nil:
		return nil, domain.ErroValidacao("empresa da dívida é obrigatória")
	case usuarioID == uuid.Nil:
		return nil, domain.ErroValidacao("usuário da dívida não informado")
	case dados.FornecedorID == uuid.Nil:
		return nil, domain.ErroValidacao("credor da dívida é obrigatório")
	}

	divida := &Divida{
		ID:           uuid.Must(uuid.NewV7()),
		EmpresaID:    empresaID,
		FornecedorID: dados.FornecedorID,
		Status:       StatusAtiva,
		UsuarioID:    usuarioID,
		Versao:       versaoInicial,
		CriadoEm:     agora,
	}
	if err := divida.aplicarInformativos(dados.DadosInformativos, agora); err != nil {
		return nil, err
	}
	if err := divida.aplicarCronograma(dados, agora); err != nil {
		return nil, err
	}

	return divida, nil
}

func (d *Divida) ConferirVersao(informada int) error {
	if informada < versaoInicial {
		return domain.ErroValidacao("versão da dívida é obrigatória")
	}
	if informada != d.Versao {
		return ErroVersaoDivergente()
	}

	return nil
}

func (d *Divida) exigirAtiva() error {
	if d.Status != StatusAtiva {
		return erroDividaEncerrada(d.Status)
	}

	return nil
}

func (d *Divida) Alterar(dados DadosInformativos, agora time.Time) error {
	if err := d.exigirAtiva(); err != nil {
		return err
	}

	return d.aplicarInformativos(dados, agora)
}

func (d *Divida) aplicarInformativos(dados DadosInformativos, agora time.Time) error {
	if dados.CredorTipo == "" {
		return domain.ErroValidacao(masculino("tipo de credor").obrigatorio())
	}
	descricao, err := textoObrigatorio(dados.Descricao, feminino("descrição da dívida"), tamanhoMaximoDescricao)
	if err != nil {
		return err
	}
	contrato, err := textoOpcional(dados.Contrato, "contrato", tamanhoMaximoContrato)
	if err != nil {
		return err
	}
	observacoes, err := textoLimitado(dados.Observacoes, "observações", tamanhoMaximoObservacoes)
	if err != nil {
		return err
	}
	if dados.ValorOriginalCentavos != nil && (*dados.ValorOriginalCentavos <= 0 || *dados.ValorOriginalCentavos > valorMaximoCentavos) {
		return domain.ErroValidacao("valor original deve ser maior que zero e no máximo 10^15 centavos")
	}
	contratadaEm, err := dataOpcionalDoFato(dados.ContratadaEm, agora, "data da contratação")
	if err != nil {
		return err
	}
	liberadaEm, err := dataOpcionalDoFato(dados.LiberadaEm, agora, "data da liberação")
	if err != nil {
		return err
	}
	if (dados.ContaLiberacaoID == nil) != (liberadaEm == nil) {
		return domain.ErroValidacao("conta e data da liberação devem ser informadas juntas")
	}
	if dados.CategoriaAmortizacaoID == uuid.Nil {
		return domain.ErroValidacao(feminino("categoria da amortização").obrigatorio())
	}
	if dados.CategoriaJurosID == uuid.Nil {
		return domain.ErroValidacao(feminino("categoria dos juros").obrigatorio())
	}
	indexador := dados.Indexador
	if indexador == "" {
		indexador = IndexadorPrefixado
	}

	d.CredorTipo = dados.CredorTipo
	d.Descricao = descricao
	d.Contrato = contrato
	d.ValorOriginalCentavos = dados.ValorOriginalCentavos
	d.ContratadaEm = contratadaEm
	d.Taxa = dados.Taxa
	d.Indexador = indexador
	d.ContaLiberacaoID = dados.ContaLiberacaoID
	d.LiberadaEm = liberadaEm
	d.CategoriaAmortizacaoID = dados.CategoriaAmortizacaoID
	d.CategoriaJurosID = dados.CategoriaJurosID
	d.Observacoes = observacoes
	d.AtualizadoEm = agora

	return nil
}

func (d *Divida) aplicarCronograma(dados DadosDivida, agora time.Time) error {
	if dados.SaldoInicialCentavos <= 0 || dados.SaldoInicialCentavos > valorMaximoCentavos {
		return domain.ErroValidacao("saldo devedor deve ser maior que zero e no máximo 10^15 centavos")
	}
	dataSaldo, err := dataDoFato(dados.DataSaldo, agora, feminino("data do saldo"))
	if err != nil {
		return err
	}
	if len(dados.Parcelas) == 0 {
		return domain.ErroValidacao("informe ao menos uma parcela que vença depois da data do saldo")
	}
	if len(dados.Parcelas) > maximoDeParcelas {
		return domain.ErroValidacao(fmt.Sprintf("a dívida aceita no máximo %d parcelas", maximoDeParcelas))
	}

	parcelas := make([]*Parcela, 0, len(dados.Parcelas))
	vencimentos := make(map[time.Time]bool, len(dados.Parcelas))
	var amortizacao int64
	for _, input := range dados.Parcelas {
		if input.Vencimento.IsZero() {
			return domain.ErroValidacao(masculino("vencimento da parcela").obrigatorio())
		}
		vencimento := DataDe(input.Vencimento)
		if !vencimento.After(dataSaldo) {
			return domain.ErroValidacao("parcela deve vencer depois da data do saldo " + formatarData(dataSaldo) +
				"; parcelas pagas ou vencidas antes dela ficam fora")
		}
		if vencimentos[vencimento] {
			return domain.ErroValidacao("duas parcelas com o vencimento " + formatarData(vencimento))
		}
		vencimentos[vencimento] = true
		if err := validarValoresDaParcela(input.AmortizacaoCentavos, input.JurosCentavos); err != nil {
			return err
		}
		amortizacao += input.AmortizacaoCentavos

		parcelas = append(parcelas, &Parcela{
			ID:                  uuid.Must(uuid.NewV7()),
			DividaID:            d.ID,
			Vencimento:          vencimento,
			AmortizacaoCentavos: input.AmortizacaoCentavos,
			JurosCentavos:       input.JurosCentavos,
			Status:              StatusParcelaPrevista,
			CriadoEm:            agora,
			AtualizadoEm:        agora,
		})
	}
	if amortizacao != dados.SaldoInicialCentavos {
		return domain.ErroValidacao(fmt.Sprintf(
			"a soma das amortizações (%s) deve ser igual ao saldo devedor na data do saldo (%s)",
			formatarReais(amortizacao), formatarReais(dados.SaldoInicialCentavos)))
	}

	sort.SliceStable(parcelas, func(i, j int) bool { return parcelas[i].Vencimento.Before(parcelas[j].Vencimento) })
	for i, parcela := range parcelas {
		parcela.Numero = i + 1
	}

	d.SaldoInicialCentavos = dados.SaldoInicialCentavos
	d.DataSaldo = dataSaldo
	d.Parcelas = parcelas

	return nil
}

func validarValoresDaParcela(amortizacao, juros int64) error {
	if amortizacao < 0 || juros < 0 {
		return domain.ErroValidacao("amortização e juros da parcela não podem ser negativos")
	}
	if amortizacao > valorMaximoCentavos || juros > valorMaximoCentavos {
		return domain.ErroValidacao("amortização e juros da parcela vão até 10^15 centavos")
	}
	if amortizacao+juros == 0 {
		return domain.ErroValidacao("parcela precisa de amortização ou juros maior que zero")
	}

	return nil
}

func (d *Divida) Parcela(id uuid.UUID) (*Parcela, error) {
	for _, parcela := range d.Parcelas {
		if parcela.ID == id {
			return parcela, nil
		}
	}

	return nil, domain.ErroNaoEncontrado("parcela da dívida não encontrada")
}

func (d *Divida) vigentes() []*Parcela {
	vigentes := make([]*Parcela, 0, len(d.Parcelas))
	for _, parcela := range d.Parcelas {
		if parcela.Status != StatusParcelaCancelada {
			vigentes = append(vigentes, parcela)
		}
	}
	sort.SliceStable(vigentes, func(i, j int) bool { return vigentes[i].Numero < vigentes[j].Numero })

	return vigentes
}

func (d *Divida) TotalDeParcelas() int {
	total := 0
	for _, parcela := range d.Parcelas {
		if parcela.Numero > total {
			total = parcela.Numero
		}
	}

	return total
}

func (d *Divida) AlterarParcela(parcelaID uuid.UUID, vencimento time.Time, juros int64, agora time.Time) (*Parcela, string, error) {
	if err := d.exigirAtiva(); err != nil {
		return nil, "", err
	}
	parcela, err := d.Parcela(parcelaID)
	if err != nil {
		return nil, "", err
	}
	if parcela.Status != StatusParcelaPrevista {
		return nil, "", domain.ErroConflito("só a parcela prevista, sem contas a pagar geradas, pode ser alterada")
	}
	if vencimento.IsZero() {
		return nil, "", domain.ErroValidacao(masculino("vencimento da parcela").obrigatorio())
	}
	if err := validarValoresDaParcela(parcela.AmortizacaoCentavos, juros); err != nil {
		return nil, "", err
	}

	novo := DataDe(vencimento)
	if !novo.After(d.DataSaldo) {
		return nil, "", domain.ErroValidacao("parcela deve vencer depois da data do saldo " + formatarData(d.DataSaldo))
	}
	vigentes := d.vigentes()
	for i, outra := range vigentes {
		if outra.ID != parcela.ID {
			continue
		}
		if i > 0 && !novo.After(vigentes[i-1].Vencimento) {
			return nil, "", domain.ErroValidacao("o vencimento deve ficar depois do da parcela " + fmt.Sprint(vigentes[i-1].Numero))
		}
		if i < len(vigentes)-1 && !novo.Before(vigentes[i+1].Vencimento) {
			return nil, "", domain.ErroValidacao("o vencimento deve ficar antes do da parcela " + fmt.Sprint(vigentes[i+1].Numero))
		}
	}

	detalhe := fmt.Sprintf("parcela %d: vencimento %s → %s, juros %s → %s", parcela.Numero,
		formatarData(parcela.Vencimento), formatarData(novo), formatarReais(parcela.JurosCentavos), formatarReais(juros))
	parcela.Vencimento = novo
	parcela.JurosCentavos = juros
	parcela.AtualizadoEm = agora
	d.AtualizadoEm = agora

	return parcela, detalhe, nil
}

func (d *Divida) Cancelar(motivo string, agora time.Time) ([]*Parcela, []ContaCancelada, error) {
	if err := d.exigirAtiva(); err != nil {
		return nil, nil, err
	}
	texto, err := textoObrigatorio(motivo, masculino("motivo do cancelamento"), tamanhoMaximoMotivo)
	if err != nil {
		return nil, nil, err
	}
	for _, parcela := range d.Parcelas {
		if parcela.Status != StatusParcelaCancelada && parcela.TemContaPaga() {
			return nil, nil, domain.ErroConflito("dívida com parcela paga não pode ser cancelada; renegocie o saldo ou estorne o pagamento antes")
		}
	}

	parcelas, contas := d.encerrarParcelas(agora)
	d.Status = StatusCancelada
	d.Motivo = &texto
	d.AtualizadoEm = agora

	return parcelas, contas, nil
}

func (d *Divida) encerrarParcelas(agora time.Time) ([]*Parcela, []ContaCancelada) {
	parcelas := make([]*Parcela, 0, len(d.Parcelas))
	contas := make([]ContaCancelada, 0)
	for _, parcela := range d.Parcelas {
		if parcela.Status == StatusParcelaCancelada {
			continue
		}
		for _, conta := range parcela.ContasAbertas() {
			contas = append(contas, ContaCancelada{SolicitacaoID: conta.SolicitacaoID, StatusAnterior: conta.Status})
			conta.Status = domainsolicitacao.StatusCancelado
		}
		if !parcela.TemContaPaga() {
			parcela.cancelar(agora)
		}
		parcelas = append(parcelas, parcela)
	}

	return parcelas, contas
}

func (d *Divida) SaldoDevedor() int64 {
	if d.Status != StatusAtiva {
		return 0
	}

	var saldo int64
	for _, parcela := range d.Parcelas {
		saldo += parcela.AmortizacaoEmAberto()
	}

	return saldo
}

func (d *Divida) Quitada() bool {
	if d.Status != StatusAtiva {
		return false
	}

	for _, parcela := range d.Parcelas {
		if parcela.Status == StatusParcelaCancelada {
			continue
		}
		if parcela.AmortizacaoEmAberto() > 0 || parcela.JurosEmAberto() > 0 {
			return false
		}
	}

	return true
}

func formatarReais(centavos int64) string {
	return fmt.Sprintf("R$ %d,%02d", centavos/100, centavos%100)
}

func (d *Divida) DadosInformativosAtuais() DadosInformativos {
	return DadosInformativos{
		CredorTipo:             d.CredorTipo,
		Descricao:              d.Descricao,
		Contrato:               d.Contrato,
		ValorOriginalCentavos:  d.ValorOriginalCentavos,
		ContratadaEm:           d.ContratadaEm,
		Taxa:                   d.Taxa,
		Indexador:              d.Indexador,
		ContaLiberacaoID:       d.ContaLiberacaoID,
		LiberadaEm:             d.LiberadaEm,
		CategoriaAmortizacaoID: d.CategoriaAmortizacaoID,
		CategoriaJurosID:       d.CategoriaJurosID,
		Observacoes:            d.Observacoes,
	}
}
