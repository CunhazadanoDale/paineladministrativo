package solicitacao_test

import (
	"bytes"
	"context"
	"io"
	"sort"
	"time"

	"github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain"
	domainsolicitacao "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/solicitacao"
	domainusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/domain/usuarios"
	portsout "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/solicitacao"
	portsoutusuarios "github.com/CunhazadanoDale/paineladministrativo.git/internal/core/ports/out/usuarios"
	"github.com/google/uuid"
)

var (
	_ portsout.SolicitacaoRepository     = (*repositorioSolicitacoes)(nil)
	_ portsout.ArquivoRepository         = (*repositorioArquivos)(nil)
	_ portsout.AprovadorRepository       = (*repositorioAprovadores)(nil)
	_ portsoutusuarios.UsuarioRepository = (*repositorioUsuarios)(nil)
	_ portsoutusuarios.CargoRepository   = (*repositorioCargos)(nil)
	_ portsout.Storage                   = (*storageFalso)(nil)
)

type repositorioSolicitacoes struct {
	itens      map[uuid.UUID]*domainsolicitacao.Solicitacao
	pagamentos map[uuid.UUID]*domainsolicitacao.Pagamento
	historicos map[uuid.UUID][]*domainsolicitacao.Historico
	arquivos   map[uuid.UUID][]*domainsolicitacao.Arquivo
	origem     *repositorioArquivos
}

func novoRepositorioSolicitacoes(origem *repositorioArquivos) *repositorioSolicitacoes {
	return &repositorioSolicitacoes{
		itens:      map[uuid.UUID]*domainsolicitacao.Solicitacao{},
		pagamentos: map[uuid.UUID]*domainsolicitacao.Pagamento{},
		historicos: map[uuid.UUID][]*domainsolicitacao.Historico{},
		arquivos:   map[uuid.UUID][]*domainsolicitacao.Arquivo{},
		origem:     origem,
	}
}

func (r *repositorioSolicitacoes) Criar(_ context.Context, solicitacao *domainsolicitacao.Solicitacao, arquivoIDs []uuid.UUID, historico *domainsolicitacao.Historico) (uuid.UUID, error) {
	copia := *solicitacao
	r.itens[solicitacao.ID] = &copia
	r.registrarHistorico(solicitacao.ID, historico)

	for _, arquivoID := range arquivoIDs {
		if arquivo, ok := r.origem.itens[arquivoID]; ok {
			r.arquivos[solicitacao.ID] = append(r.arquivos[solicitacao.ID], arquivo)
			r.origem.vinculos[arquivoID] = solicitacao.ID
		}
	}

	return solicitacao.ID, nil
}

func (r *repositorioSolicitacoes) Obter(_ context.Context, id uuid.UUID) (*domainsolicitacao.Solicitacao, error) {
	solicitacao, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *solicitacao
	return &copia, nil
}

func (r *repositorioSolicitacoes) Listar(_ context.Context, filtro portsout.SolicitacaoFiltro) ([]*domainsolicitacao.Solicitacao, error) {
	coincidindo := make([]*domainsolicitacao.Solicitacao, 0)

	for _, solicitacao := range r.itens {
		if filtro.Solicitante != nil && solicitacao.SolicitanteID != *filtro.Solicitante {
			continue
		}
		if filtro.Status != "" && solicitacao.Status != filtro.Status {
			continue
		}
		copia := *solicitacao
		coincidindo = append(coincidindo, &copia)
	}

	sort.SliceStable(coincidindo, func(i, j int) bool {
		return coincidindo[i].CriadoEm.After(coincidindo[j].CriadoEm)
	})

	return paginar(coincidindo, filtro.PaginacaoFiltro), nil
}

func (r *repositorioSolicitacoes) AtualizarStatus(_ context.Context, solicitacao *domainsolicitacao.Solicitacao, historico *domainsolicitacao.Historico) (bool, error) {
	salvo, ok := r.itens[solicitacao.ID]
	if !ok || historico.DeStatus == nil || salvo.Status != *historico.DeStatus {
		return false, nil
	}

	copia := *solicitacao
	r.itens[solicitacao.ID] = &copia
	r.registrarHistorico(solicitacao.ID, historico)

	return true, nil
}

func (r *repositorioSolicitacoes) CriarPagamento(_ context.Context, solicitacao *domainsolicitacao.Solicitacao, pagamento *domainsolicitacao.Pagamento, historico *domainsolicitacao.Historico) error {
	atualizado, err := r.AtualizarStatus(context.Background(), solicitacao, historico)
	if err != nil {
		return err
	}
	if !atualizado {
		return domain.ErroConflito("solicitação alterada por outra operação")
	}

	r.pagamentos[solicitacao.ID] = pagamento

	return nil
}

func (r *repositorioSolicitacoes) ListarArquivos(_ context.Context, solicitacaoID uuid.UUID) ([]*domainsolicitacao.Arquivo, error) {
	return r.arquivos[solicitacaoID], nil
}

func (r *repositorioSolicitacoes) ListarHistorico(_ context.Context, solicitacaoID uuid.UUID) ([]*domainsolicitacao.Historico, error) {
	registros := r.historicos[solicitacaoID]
	itens := make([]*domainsolicitacao.Historico, 0, len(registros))

	for posicao := len(registros) - 1; posicao >= 0; posicao-- {
		itens = append(itens, registros[posicao])
	}

	return itens, nil
}

func (r *repositorioSolicitacoes) registrarHistorico(solicitacaoID uuid.UUID, historico *domainsolicitacao.Historico) {
	r.historicos[solicitacaoID] = append(r.historicos[solicitacaoID], historico)
}

func (r *repositorioSolicitacoes) ObterPagamento(_ context.Context, solicitacaoID uuid.UUID) (*domainsolicitacao.Pagamento, error) {
	return r.pagamentos[solicitacaoID], nil
}

type repositorioArquivos struct {
	itens         map[uuid.UUID]*domainsolicitacao.Arquivo
	vinculos      map[uuid.UUID]uuid.UUID
	falharCriacao bool
}

func novoRepositorioArquivos() *repositorioArquivos {
	return &repositorioArquivos{
		itens:    map[uuid.UUID]*domainsolicitacao.Arquivo{},
		vinculos: map[uuid.UUID]uuid.UUID{},
	}
}

func (r *repositorioArquivos) Criar(_ context.Context, arquivo *domainsolicitacao.Arquivo) (uuid.UUID, error) {
	if r.falharCriacao {
		return uuid.Nil, domain.ErroValidacao("falha simulada no repositório de arquivos")
	}

	copia := *arquivo
	r.itens[arquivo.ID] = &copia

	return arquivo.ID, nil
}

func (r *repositorioArquivos) Obter(_ context.Context, id uuid.UUID) (*domainsolicitacao.Arquivo, error) {
	arquivo, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *arquivo
	return &copia, nil
}

func (r *repositorioArquivos) ListarPorProprietario(_ context.Context, proprietarioID uuid.UUID, filtro domain.PaginacaoFiltro) ([]*domainsolicitacao.Arquivo, error) {
	coincidindo := make([]*domainsolicitacao.Arquivo, 0)

	for _, arquivo := range r.itens {
		if arquivo.ProprietarioID != proprietarioID {
			continue
		}
		copia := *arquivo
		coincidindo = append(coincidindo, &copia)
	}

	sort.SliceStable(coincidindo, func(i, j int) bool {
		return coincidindo[i].CriadoEm.After(coincidindo[j].CriadoEm)
	})

	return paginar(coincidindo, filtro), nil
}

func (r *repositorioArquivos) VinculadoASolicitacao(_ context.Context, arquivoID uuid.UUID) (bool, error) {
	_, vinculado := r.vinculos[arquivoID]

	return vinculado, nil
}

func (r *repositorioArquivos) SolicitacaoDoArquivo(_ context.Context, arquivoID uuid.UUID) (*uuid.UUID, error) {
	solicitacaoID, vinculado := r.vinculos[arquivoID]
	if !vinculado {
		return nil, nil
	}

	return &solicitacaoID, nil
}

func (r *repositorioArquivos) Remover(_ context.Context, id uuid.UUID) error {
	delete(r.itens, id)
	delete(r.vinculos, id)

	return nil
}

type repositorioAprovadores struct {
	itens map[uuid.UUID]*domainsolicitacao.Aprovador
}

func novoRepositorioAprovadores() *repositorioAprovadores {
	return &repositorioAprovadores{itens: map[uuid.UUID]*domainsolicitacao.Aprovador{}}
}

func (r *repositorioAprovadores) Criar(_ context.Context, aprovador *domainsolicitacao.Aprovador) (uuid.UUID, error) {
	copia := *aprovador
	r.itens[aprovador.ID] = &copia

	return aprovador.ID, nil
}

func (r *repositorioAprovadores) Obter(_ context.Context, id uuid.UUID) (*domainsolicitacao.Aprovador, error) {
	aprovador, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *aprovador
	return &copia, nil
}

func (r *repositorioAprovadores) ObterPorUsuarioID(_ context.Context, usuarioID uuid.UUID) (*domainsolicitacao.Aprovador, error) {
	for _, aprovador := range r.itens {
		if aprovador.UsuarioID == usuarioID {
			copia := *aprovador
			return &copia, nil
		}
	}

	return nil, nil
}

func (r *repositorioAprovadores) Listar(_ context.Context, filtro domain.PaginacaoFiltro) ([]*domainsolicitacao.Aprovador, error) {
	itens := make([]*domainsolicitacao.Aprovador, 0, len(r.itens))
	for _, aprovador := range r.itens {
		copia := *aprovador
		itens = append(itens, &copia)
	}

	sort.SliceStable(itens, func(i, j int) bool {
		return itens[i].CriadoEm.Before(itens[j].CriadoEm)
	})

	return paginar(itens, filtro), nil
}

func (r *repositorioAprovadores) Remover(_ context.Context, id uuid.UUID) error {
	delete(r.itens, id)

	return nil
}

type repositorioUsuarios struct {
	itens map[uuid.UUID]*domainusuarios.Usuario
}

func novoRepositorioUsuarios() *repositorioUsuarios {
	return &repositorioUsuarios{itens: map[uuid.UUID]*domainusuarios.Usuario{}}
}

func (r *repositorioUsuarios) Create(_ context.Context, usuario *domainusuarios.Usuario) (uuid.UUID, error) {
	copia := *usuario
	r.itens[usuario.ID] = &copia

	return usuario.ID, nil
}

func (r *repositorioUsuarios) Update(_ context.Context, usuario *domainusuarios.Usuario) error {
	copia := *usuario
	r.itens[usuario.ID] = &copia

	return nil
}

func (r *repositorioUsuarios) GetByID(_ context.Context, id uuid.UUID) (*domainusuarios.Usuario, error) {
	usuario, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *usuario
	return &copia, nil
}

func (r *repositorioUsuarios) GetByEmail(_ context.Context, email string) (*domainusuarios.Usuario, error) {
	for _, usuario := range r.itens {
		if usuario.Email == email {
			copia := *usuario
			return &copia, nil
		}
	}

	return nil, nil
}

func (r *repositorioUsuarios) List(_ context.Context, _ domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	return nil, nil
}

func (r *repositorioUsuarios) ListAtivos(_ context.Context, _ domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	return nil, nil
}

func (r *repositorioUsuarios) Search(_ context.Context, _ string, _ domain.PaginacaoFiltro) ([]*domainusuarios.Usuario, error) {
	return nil, nil
}

func (r *repositorioUsuarios) UpdateUltimoLogin(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

func (r *repositorioUsuarios) AtualizarSenha(_ context.Context, _ uuid.UUID, _ string, _ time.Time) error {
	return nil
}

func (r *repositorioUsuarios) EncerrarSessoes(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (r *repositorioUsuarios) Ativar(_ context.Context, id uuid.UUID) error {
	if usuario, ok := r.itens[id]; ok {
		usuario.Ativo = true
	}

	return nil
}

func (r *repositorioUsuarios) Desativar(_ context.Context, id uuid.UUID) error {
	if usuario, ok := r.itens[id]; ok {
		usuario.Ativo = false
	}

	return nil
}

func (r *repositorioUsuarios) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.itens, id)

	return nil
}

type repositorioCargos struct {
	itens map[uuid.UUID]*domainusuarios.Cargo
}

func novoRepositorioCargos() *repositorioCargos {
	return &repositorioCargos{itens: map[uuid.UUID]*domainusuarios.Cargo{}}
}

func (r *repositorioCargos) Create(_ context.Context, cargo *domainusuarios.Cargo) (uuid.UUID, error) {
	copia := *cargo
	r.itens[cargo.ID] = &copia

	return cargo.ID, nil
}

func (r *repositorioCargos) Update(_ context.Context, cargo *domainusuarios.Cargo) error {
	copia := *cargo
	r.itens[cargo.ID] = &copia

	return nil
}

func (r *repositorioCargos) GetByID(_ context.Context, id uuid.UUID) (*domainusuarios.Cargo, error) {
	cargo, ok := r.itens[id]
	if !ok {
		return nil, nil
	}

	copia := *cargo
	return &copia, nil
}

func (r *repositorioCargos) GetByNome(_ context.Context, nome string) (*domainusuarios.Cargo, error) {
	for _, cargo := range r.itens {
		if cargo.Nome == nome {
			copia := *cargo
			return &copia, nil
		}
	}

	return nil, nil
}

func (r *repositorioCargos) List(_ context.Context, _ domain.PaginacaoFiltro) ([]*domainusuarios.Cargo, error) {
	return nil, nil
}

func (r *repositorioCargos) ListAtivos(_ context.Context, _ domain.PaginacaoFiltro) ([]*domainusuarios.Cargo, error) {
	return nil, nil
}

func (r *repositorioCargos) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.itens, id)

	return nil
}

type storageFalso struct {
	conteudos map[string][]byte
	tipos     map[string]string
	falhar    bool
}

func novoStorageFalso() *storageFalso {
	return &storageFalso{
		conteudos: map[string][]byte{},
		tipos:     map[string]string{},
	}
}

func (s *storageFalso) Enviar(_ context.Context, chave string, conteudo io.Reader, contentType string) error {
	if s.falhar {
		return domain.ErroValidacao("falha simulada no storage")
	}

	dados, err := io.ReadAll(conteudo)
	if err != nil {
		return err
	}

	s.conteudos[chave] = dados
	s.tipos[chave] = contentType

	return nil
}

func (s *storageFalso) Baixar(_ context.Context, chave string) (io.ReadCloser, error) {
	dados, ok := s.conteudos[chave]
	if !ok {
		return nil, domain.ErrNotFound
	}

	return io.NopCloser(bytes.NewReader(dados)), nil
}

func (s *storageFalso) Remover(_ context.Context, chave string) error {
	delete(s.conteudos, chave)
	delete(s.tipos, chave)

	return nil
}

func paginar[T any](itens []T, filtro domain.PaginacaoFiltro) []T {
	normalizado := filtro.Normalizada()
	inicio := (normalizado.Page - 1) * normalizado.Size
	if inicio >= len(itens) {
		return []T{}
	}

	fim := inicio + normalizado.Size
	if fim > len(itens) {
		fim = len(itens)
	}

	return itens[inicio:fim]
}
