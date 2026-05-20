package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"AIM/internal/model"

	"gorm.io/gorm"
)

const maxKnowledgeContextChars = 12000

type AIKnowledgeBaseInput struct {
	Name        string
	Description string
}

type AIKnowledgeDocumentInput struct {
	Title   string
	Content string
}

type AIKnowledgeDocumentInfo struct {
	ID              uint      `json:"id"`
	KnowledgeBaseID uint      `json:"knowledge_base_id"`
	Title           string    `json:"title"`
	Content         string    `json:"content"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type AIKnowledgeBaseInfo struct {
	ID            uint                      `json:"id"`
	Name          string                    `json:"name"`
	Description   string                    `json:"description"`
	DocumentCount int                       `json:"document_count"`
	Documents     []AIKnowledgeDocumentInfo `json:"documents"`
	CreatedAt     time.Time                 `json:"created_at"`
	UpdatedAt     time.Time                 `json:"updated_at"`
}

func (s *AIService) ListKnowledgeBases(ownerID uint) ([]AIKnowledgeBaseInfo, error) {
	var bases []model.AIKnowledgeBase
	if err := model.DB.
		Where("owner_id = ? AND is_system = ? AND status <> ?", ownerID, false, model.AIKnowledgeStatusDeleted).
		Order("updated_at DESC").
		Find(&bases).Error; err != nil {
		return nil, err
	}

	result := make([]AIKnowledgeBaseInfo, 0, len(bases))
	for _, base := range bases {
		docs, err := s.listKnowledgeDocuments(ownerID, base.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, toAIKnowledgeBaseInfo(base, docs))
	}
	return result, nil
}

func (s *AIService) CreateKnowledgeBase(ownerID uint, input AIKnowledgeBaseInput) (*AIKnowledgeBaseInfo, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	if input.Name == "" {
		return nil, errors.New("知识库名称不能为空")
	}

	base := model.AIKnowledgeBase{
		OwnerID:     &ownerID,
		Name:        input.Name,
		Description: input.Description,
		IsSystem:    false,
		Status:      model.AIKnowledgeStatusEnabled,
	}
	if err := model.DB.Create(&base).Error; err != nil {
		return nil, err
	}
	info := toAIKnowledgeBaseInfo(base, nil)
	return &info, nil
}

func (s *AIService) UpdateKnowledgeBase(ownerID, baseID uint, input AIKnowledgeBaseInput) (*AIKnowledgeBaseInfo, error) {
	base, err := s.getOwnedKnowledgeBase(ownerID, baseID)
	if err != nil {
		return nil, err
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	if input.Name == "" {
		return nil, errors.New("知识库名称不能为空")
	}
	if err := model.DB.Model(base).Updates(map[string]interface{}{
		"name":        input.Name,
		"description": input.Description,
	}).Error; err != nil {
		return nil, err
	}
	docs, err := s.listKnowledgeDocuments(ownerID, baseID)
	if err != nil {
		return nil, err
	}
	base.Name = input.Name
	base.Description = input.Description
	info := toAIKnowledgeBaseInfo(*base, docs)
	return &info, nil
}

func (s *AIService) DeleteKnowledgeBase(ownerID, baseID uint) error {
	if _, err := s.getOwnedKnowledgeBase(ownerID, baseID); err != nil {
		return err
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.AIKnowledgeBase{}).
			Where("id = ?", baseID).
			Update("status", model.AIKnowledgeStatusDeleted).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.AIKnowledgeDocument{}).
			Where("knowledge_base_id = ?", baseID).
			Update("status", model.AIKnowledgeStatusDeleted).Error; err != nil {
			return err
		}
		return tx.Where("knowledge_base_id = ?", baseID).Delete(&model.AIBotKnowledgeBase{}).Error
	})
}

func (s *AIService) AddKnowledgeDocument(ownerID, baseID uint, input AIKnowledgeDocumentInput) (*AIKnowledgeDocumentInfo, error) {
	if _, err := s.getOwnedKnowledgeBase(ownerID, baseID); err != nil {
		return nil, err
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	if input.Title == "" {
		return nil, errors.New("文档标题不能为空")
	}
	if input.Content == "" {
		return nil, errors.New("文档内容不能为空")
	}
	doc := model.AIKnowledgeDocument{
		KnowledgeBaseID: baseID,
		Title:           input.Title,
		Content:         input.Content,
		Status:          model.AIKnowledgeStatusEnabled,
	}
	if err := model.DB.Create(&doc).Error; err != nil {
		return nil, err
	}
	info := toAIKnowledgeDocumentInfo(doc)
	return &info, nil
}

func (s *AIService) UpdateKnowledgeDocument(ownerID, docID uint, input AIKnowledgeDocumentInput) (*AIKnowledgeDocumentInfo, error) {
	doc, err := s.getOwnedKnowledgeDocument(ownerID, docID)
	if err != nil {
		return nil, err
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)
	if input.Title == "" {
		return nil, errors.New("文档标题不能为空")
	}
	if input.Content == "" {
		return nil, errors.New("文档内容不能为空")
	}
	if err := model.DB.Model(doc).Updates(map[string]interface{}{
		"title":   input.Title,
		"content": input.Content,
	}).Error; err != nil {
		return nil, err
	}
	doc.Title = input.Title
	doc.Content = input.Content
	info := toAIKnowledgeDocumentInfo(*doc)
	return &info, nil
}

func (s *AIService) DeleteKnowledgeDocument(ownerID, docID uint) error {
	if _, err := s.getOwnedKnowledgeDocument(ownerID, docID); err != nil {
		return err
	}
	return model.DB.Model(&model.AIKnowledgeDocument{}).
		Where("id = ?", docID).
		Update("status", model.AIKnowledgeStatusDeleted).Error
}

func (s *AIService) replaceBotKnowledgeBasesTx(tx *gorm.DB, ownerID, botID uint, ids []uint) error {
	ids = uniqueUintIDs(ids)
	if len(ids) > 0 {
		var count int64
		if err := tx.Model(&model.AIKnowledgeBase{}).
			Where("id IN ? AND owner_id = ? AND is_system = ? AND status <> ?", ids, ownerID, false, model.AIKnowledgeStatusDeleted).
			Count(&count).Error; err != nil {
			return err
		}
		if count != int64(len(ids)) {
			return errors.New("知识库不存在或无权使用")
		}
	}
	if err := tx.Where("ai_bot_id = ?", botID).Delete(&model.AIBotKnowledgeBase{}).Error; err != nil {
		return err
	}
	for _, id := range ids {
		if err := tx.Create(&model.AIBotKnowledgeBase{
			AIBotID:         botID,
			KnowledgeBaseID: id,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *AIService) listBotKnowledgeBaseIDs(ownerID uint, bot model.AIBot) ([]uint, error) {
	if bot.IsSystem {
		return nil, nil
	}
	var ids []uint
	err := model.DB.Table("ai_bot_knowledge_bases").
		Select("ai_bot_knowledge_bases.knowledge_base_id").
		Joins("JOIN ai_knowledge_bases ON ai_knowledge_bases.id = ai_bot_knowledge_bases.knowledge_base_id").
		Where("ai_bot_knowledge_bases.ai_bot_id = ? AND ai_knowledge_bases.owner_id = ? AND ai_knowledge_bases.is_system = ? AND ai_knowledge_bases.status <> ?",
			bot.ID, ownerID, false, model.AIKnowledgeStatusDeleted).
		Order("ai_bot_knowledge_bases.id ASC").
		Pluck("ai_bot_knowledge_bases.knowledge_base_id", &ids).Error
	return ids, err
}

func (s *AIService) loadKnowledgeContext(runtime *aiRuntime) (string, error) {
	if runtime == nil || runtime.Bot == nil {
		return "", nil
	}
	type knowledgeRow struct {
		BaseName string
		Title    string
		Content  string
	}
	var rows []knowledgeRow
	if err := model.DB.Table("ai_knowledge_documents").
		Select("ai_knowledge_bases.name AS base_name, ai_knowledge_documents.title, ai_knowledge_documents.content").
		Joins("JOIN ai_bot_knowledge_bases ON ai_bot_knowledge_bases.knowledge_base_id = ai_knowledge_documents.knowledge_base_id").
		Joins("JOIN ai_knowledge_bases ON ai_knowledge_bases.id = ai_knowledge_documents.knowledge_base_id").
		Where("ai_bot_knowledge_bases.ai_bot_id = ? AND ai_knowledge_bases.status = ? AND ai_knowledge_documents.status = ?",
			runtime.Bot.ID, model.AIKnowledgeStatusEnabled, model.AIKnowledgeStatusEnabled).
		Order("ai_knowledge_bases.id ASC, ai_knowledge_documents.id ASC").
		Limit(30).
		Scan(&rows).Error; err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", nil
	}

	var builder strings.Builder
	for _, row := range rows {
		content := strings.TrimSpace(row.Content)
		if content == "" {
			continue
		}
		entry := fmt.Sprintf("知识库：%s\n标题：%s\n内容：%s\n\n",
			strings.TrimSpace(row.BaseName),
			strings.TrimSpace(row.Title),
			content,
		)
		if builder.Len()+len(entry) > maxKnowledgeContextChars {
			remaining := maxKnowledgeContextChars - builder.Len()
			if remaining > 0 {
				builder.WriteString(entry[:remaining])
			}
			break
		}
		builder.WriteString(entry)
	}
	return strings.TrimSpace(builder.String()), nil
}

func (s *AIService) listKnowledgeDocuments(ownerID, baseID uint) ([]AIKnowledgeDocumentInfo, error) {
	if _, err := s.getOwnedKnowledgeBase(ownerID, baseID); err != nil {
		return nil, err
	}
	var docs []model.AIKnowledgeDocument
	if err := model.DB.
		Where("knowledge_base_id = ? AND status <> ?", baseID, model.AIKnowledgeStatusDeleted).
		Order("updated_at DESC").
		Find(&docs).Error; err != nil {
		return nil, err
	}
	result := make([]AIKnowledgeDocumentInfo, 0, len(docs))
	for _, doc := range docs {
		result = append(result, toAIKnowledgeDocumentInfo(doc))
	}
	return result, nil
}

func (s *AIService) getOwnedKnowledgeBase(ownerID, baseID uint) (*model.AIKnowledgeBase, error) {
	var base model.AIKnowledgeBase
	if err := model.DB.
		Where("id = ? AND owner_id = ? AND is_system = ? AND status <> ?", baseID, ownerID, false, model.AIKnowledgeStatusDeleted).
		First(&base).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("知识库不存在或无权操作")
		}
		return nil, err
	}
	return &base, nil
}

func (s *AIService) getOwnedKnowledgeDocument(ownerID, docID uint) (*model.AIKnowledgeDocument, error) {
	var doc model.AIKnowledgeDocument
	err := model.DB.
		Joins("JOIN ai_knowledge_bases ON ai_knowledge_bases.id = ai_knowledge_documents.knowledge_base_id").
		Where("ai_knowledge_documents.id = ? AND ai_knowledge_bases.owner_id = ? AND ai_knowledge_bases.is_system = ? AND ai_knowledge_bases.status <> ? AND ai_knowledge_documents.status <> ?",
			docID, ownerID, false, model.AIKnowledgeStatusDeleted, model.AIKnowledgeStatusDeleted).
		First(&doc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("知识库文档不存在或无权操作")
		}
		return nil, err
	}
	return &doc, nil
}

func toAIKnowledgeBaseInfo(base model.AIKnowledgeBase, docs []AIKnowledgeDocumentInfo) AIKnowledgeBaseInfo {
	return AIKnowledgeBaseInfo{
		ID:            base.ID,
		Name:          base.Name,
		Description:   base.Description,
		DocumentCount: len(docs),
		Documents:     docs,
		CreatedAt:     base.CreatedAt,
		UpdatedAt:     base.UpdatedAt,
	}
}

func toAIKnowledgeDocumentInfo(doc model.AIKnowledgeDocument) AIKnowledgeDocumentInfo {
	return AIKnowledgeDocumentInfo{
		ID:              doc.ID,
		KnowledgeBaseID: doc.KnowledgeBaseID,
		Title:           doc.Title,
		Content:         doc.Content,
		Status:          doc.Status,
		CreatedAt:       doc.CreatedAt,
		UpdatedAt:       doc.UpdatedAt,
	}
}
