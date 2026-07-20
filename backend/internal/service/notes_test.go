package service

import (
	"context"
	"errors"
	"testing"

	"github.com/morfostech/morfos-finance/internal/domain"
)

// fakeNoteRepo is an in-memory NoteRepo enforcing the same ownership rule the
// real repository enforces via SQL WHERE user_id = $owner.
type fakeNoteRepo struct {
	store  map[int64]*domain.Note
	nextID int64
}

func newFakeNoteRepo() *fakeNoteRepo {
	return &fakeNoteRepo{store: map[int64]*domain.Note{}, nextID: 1}
}

func (f *fakeNoteRepo) Create(_ context.Context, n *domain.Note) (*domain.Note, error) {
	n.ID = f.nextID
	f.nextID++
	cp := *n
	f.store[n.ID] = &cp
	return n, nil
}

func (f *fakeNoteRepo) List(_ context.Context, userID int64, ownerType domain.NoteOwner, ownerID *int64) ([]domain.Note, error) {
	var out []domain.Note
	for _, n := range f.store {
		if n.UserID == userID && n.OwnerType == ownerType && ptrEqual(n.OwnerID, ownerID) {
			out = append(out, *n)
		}
	}
	return out, nil
}

func (f *fakeNoteRepo) GetOwned(_ context.Context, id, userID int64) (*domain.Note, error) {
	n, ok := f.store[id]
	if !ok || n.UserID != userID {
		return nil, domain.ErrNotFound
	}
	return n, nil
}

func (f *fakeNoteRepo) Update(_ context.Context, id, userID int64, texto string) (*domain.Note, error) {
	n, ok := f.store[id]
	if !ok || n.UserID != userID {
		return nil, domain.ErrNotFound
	}
	n.Texto = texto
	return n, nil
}

func (f *fakeNoteRepo) Delete(_ context.Context, id, userID int64) error {
	n, ok := f.store[id]
	if !ok || n.UserID != userID {
		return domain.ErrNotFound
	}
	delete(f.store, id)
	return nil
}

func ptrEqual(a, b *int64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func TestNoteCreateValidation(t *testing.T) {
	pid := int64(5)
	tests := []struct {
		name      string
		ownerType domain.NoteOwner
		ownerID   *int64
		texto     string
		wantErr   error
	}{
		{"geral sem owner_id", domain.NoteOwnerGeral, nil, "lembrar de cobrar", nil},
		{"projeto com owner_id", domain.NoteOwnerProject, &pid, "negociar desconto", nil},
		{"projeto sem owner_id", domain.NoteOwnerProject, nil, "x", domain.ErrValidation},
		{"tipo inválido", domain.NoteOwner("outro"), &pid, "x", domain.ErrValidation},
		{"texto vazio", domain.NoteOwnerGeral, nil, "   ", domain.ErrValidation},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewNoteService(newFakeNoteRepo())
			_, err := svc.Create(context.Background(), 1, tc.ownerType, tc.ownerID, tc.texto)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestNotesAreUserScoped(t *testing.T) {
	repo := newFakeNoteRepo()
	svc := NewNoteService(repo)
	ctx := context.Background()
	pid := int64(1)

	noteA, err := svc.Create(ctx, 10, domain.NoteOwnerProject, &pid, "nota da Ana")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(ctx, 20, domain.NoteOwnerProject, &pid, "nota do Bruno"); err != nil {
		t.Fatal(err)
	}

	// Cada um só lista a própria nota no mesmo projeto.
	listaAna, err := svc.List(ctx, 10, domain.NoteOwnerProject, &pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(listaAna) != 1 || listaAna[0].Texto != "nota da Ana" {
		t.Fatalf("Ana deveria ver só a própria nota, veio: %+v", listaAna)
	}

	// Bruno não consegue editar a nota da Ana.
	if _, err := svc.Update(ctx, noteA.ID, 20, "tentando editar"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound (nota de outro usuário deve parecer inexistente)", err)
	}

	// Bruno não consegue apagar a nota da Ana.
	if err := svc.Delete(ctx, noteA.ID, 20); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}

	// Ana edita a própria nota normalmente.
	updated, err := svc.Update(ctx, noteA.ID, 10, "nota atualizada")
	if err != nil {
		t.Fatalf("Ana deveria conseguir editar a própria nota: %v", err)
	}
	if updated.Texto != "nota atualizada" {
		t.Errorf("texto = %q, want %q", updated.Texto, "nota atualizada")
	}
}

func TestNoteUpdateEmptyTexto(t *testing.T) {
	repo := newFakeNoteRepo()
	svc := NewNoteService(repo)
	ctx := context.Background()

	n, _ := svc.Create(ctx, 1, domain.NoteOwnerGeral, nil, "original")
	if _, err := svc.Update(ctx, n.ID, 1, "   "); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
}
