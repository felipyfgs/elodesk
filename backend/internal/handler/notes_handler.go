package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
	"backend/internal/service"
)

type NotesHandler struct {
	svc *service.NotesService
}

func NewNotesHandler(svc *service.NotesService) *NotesHandler {
	return &NotesHandler{svc: svc}
}

func (h *NotesHandler) List(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	contactID, err := strconv.ParseInt(c.Params("cid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid contact id"))
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "25"))

	notes, total, err := h.svc.ListByContact(c.Context(), contactID, accountID, page, perPage)
	if err != nil {
		return handleNotFound(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.NoteListResp{
		Meta:    dto.NewMetaResp(total, page, perPage),
		Payload: dto.NotesToResp(notes),
	}))
}

func (h *NotesHandler) Create(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	contactID, err := strconv.ParseInt(c.Params("cid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid contact id"))
	}

	user, ok := c.Locals("user").(*repo.AuthUser)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "user not found"))
	}

	var req dto.CreateNoteReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	note, err := h.svc.Create(c.Context(), accountID, contactID, user.ID, req.Content)
	if err != nil {
		return handleNoteError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(dto.SuccessResp(dto.NoteToResp(note)))
}

func (h *NotesHandler) Update(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	user, ok := c.Locals("user").(*repo.AuthUser)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "user not found"))
	}

	role := model.RoleAgent
	if r, ok := c.Locals("role").(model.Role); ok {
		role = r
	}

	nid, err := strconv.ParseInt(c.Params("nid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid note id"))
	}

	var req dto.UpdateNoteReq
	if err := parseAndValidate(c, &req); err != nil {
		return nil
	}

	note, err := h.svc.Update(c.Context(), nid, accountID, user.ID, int(role), req.Content)
	if err != nil {
		return handleNoteError(c, err)
	}

	return c.JSON(dto.SuccessResp(dto.NoteToResp(note)))
}

func (h *NotesHandler) Delete(c *fiber.Ctx) error {
	accountID, ok := c.Locals("accountId").(int64)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "account id not found"))
	}

	user, ok := c.Locals("user").(*repo.AuthUser)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "user not found"))
	}

	role := model.RoleAgent
	if r, ok := c.Locals("role").(model.Role); ok {
		role = r
	}

	nid, err := strconv.ParseInt(c.Params("nid"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(dto.ErrorResp("Bad Request", "invalid note id"))
	}

	if err := h.svc.Delete(c.Context(), nid, accountID, user.ID, int(role)); err != nil {
		return handleNoteError(c, err)
	}

	return c.JSON(dto.SuccessResp(map[string]string{"result": "success"}))
}

func handleNoteError(c *fiber.Ctx, err error) error {
	switch {
	case repo.IsErrNotFound(err):
		return c.Status(fiber.StatusNotFound).JSON(dto.ErrorResp("Not Found", "note not found"))
	case err == service.ErrNotNoteOwner:
		return c.Status(fiber.StatusForbidden).JSON(dto.ErrorResp("Forbidden", "not_note_owner"))
	default:
		logger.Error().Str("component", "notes").Err(err).Msg("notes service error")
		return c.Status(fiber.StatusInternalServerError).JSON(dto.ErrorResp("Error", "internal server error"))
	}
}
