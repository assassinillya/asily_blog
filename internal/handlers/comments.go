package handlers

import (
	"asily_blog/internal/models"
	"asily_blog/internal/utils"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type addCommentPayload struct {
	BlogID   string `json:"blogId"`
	ParentID string `json:"parentId"`
	Content  string `json:"content"`
	QQ       string `json:"qq"`
	Username string `json:"username"`
}

type commentResponse struct {
	ID              string    `json:"_id"`
	BlogID          string    `json:"blogId"`
	ParentID        string    `json:"parentId,omitempty"`
	RootID          string    `json:"rootId,omitempty"`
	ReplyToUsername string    `json:"replyToUsername,omitempty"`
	ReplyPrefix     string    `json:"replyPrefix,omitempty"`
	Username        string    `json:"username"`
	QQ              string    `json:"qq"`
	Content         string    `json:"content"`
	DisplayContent  string    `json:"displayContent"`
	CreatedAt       time.Time `json:"createdAt"`
	Like            int       `json:"like"`
	LikeCount       int       `json:"likeCount"`
	UnLikeCount     int       `json:"unlikeCount"`
	IsFloorOwner    bool      `json:"isFloorOwner"`
}

type floorCommentResponse struct {
	commentResponse
	Replies    []commentResponse `json:"replies"`
	ReplyCount int               `json:"replyCount"`
}

// AddComment 添加评论或层内回复
func AddComment(c *gin.Context) {
	var data addCommentPayload
	if err := c.ShouldBindJSON(&data); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	data.BlogID = strings.TrimSpace(data.BlogID)
	data.ParentID = strings.TrimSpace(data.ParentID)
	data.Content = strings.TrimSpace(data.Content)
	data.Username = strings.TrimSpace(data.Username)
	data.QQ = strings.TrimSpace(data.QQ)

	if data.BlogID == "" || data.Content == "" || data.Username == "" {
		respondError(c, http.StatusBadRequest, "博客ID、内容和用户名不能为空")
		return
	}

	blogID, err := primitive.ObjectIDFromHex(data.BlogID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的博客ID格式")
		return
	}

	blogsDB := utils.GetCollection("blogs")
	count, err := blogsDB.CountDocuments(context.Background(), bson.M{"_id": blogID})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "检查博客是否存在时出错: "+err.Error())
		return
	}
	if count == 0 {
		respondError(c, http.StatusNotFound, "评论关联的博客不存在")
		return
	}

	comment := models.Comment{
		BlogID:      blogID,
		Username:    data.Username,
		QQ:          data.QQ,
		Content:     data.Content,
		CreatedAt:   time.Now(),
		Like:        0,
		LikeCount:   0,
		UnLikeCount: 0,
	}

	isFloorOwner := true
	if data.ParentID != "" {
		parentID, err := primitive.ObjectIDFromHex(data.ParentID)
		if err != nil {
			respondError(c, http.StatusBadRequest, "无效的父评论ID格式")
			return
		}

		parent, err := findCommentByID(parentID)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				respondError(c, http.StatusNotFound, "回复目标评论不存在")
				return
			}
			respondError(c, http.StatusInternalServerError, "查询父评论失败: "+err.Error())
			return
		}
		if parent.BlogID != blogID {
			respondError(c, http.StatusBadRequest, "父评论不属于当前博客")
			return
		}

		rootID := resolveRootCommentID(parent)
		comment.ParentID = &parentID
		comment.RootID = &rootID
		comment.ReplyToUsername = parent.Username
		isFloorOwner = false
	}

	commentsDB := utils.GetCollection("comments")
	result, err := commentsDB.InsertOne(context.Background(), comment)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "评论插入失败: "+err.Error())
		return
	}

	insertedID, _ := result.InsertedID.(primitive.ObjectID)
	response := gin.H{
		"id":           insertedID.Hex(),
		"isFloorOwner": isFloorOwner,
	}
	if comment.ParentID != nil {
		response["parentId"] = comment.ParentID.Hex()
	}
	if comment.RootID != nil {
		response["rootId"] = comment.RootID.Hex()
		response["replyToUsername"] = comment.ReplyToUsername
		response["replyPrefix"] = buildReplyPrefix(comment.ReplyToUsername)
	}

	respondMessageData(c, http.StatusOK, "评论插入成功", response)
}

// ResetComments 编辑评论
func ResetComments(c *gin.Context) {
	var data struct {
		ID      string `json:"_id"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	data.Content = strings.TrimSpace(data.Content)
	if data.Content == "" {
		respondError(c, http.StatusBadRequest, "评论内容不能为空")
		return
	}

	id, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的评论ID格式")
		return
	}

	db := utils.GetCollection("comments")
	result, err := db.UpdateByID(context.Background(), id, bson.M{
		"$set": bson.M{"content": data.Content},
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "评论编辑失败: "+err.Error())
		return
	}

	if result.MatchedCount == 0 {
		respondError(c, http.StatusNotFound, "找不到要编辑的评论")
		return
	}

	respondMessage(c, http.StatusOK, "评论编辑成功")
}

// DeleteComments 删除评论
func DeleteComments(c *gin.Context) {
	var data struct {
		ID string `json:"_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	id, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的评论ID格式")
		return
	}

	comment, err := findCommentByID(id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			respondError(c, http.StatusNotFound, "找不到要删除的评论")
			return
		}
		respondError(c, http.StatusInternalServerError, "查询评论失败: "+err.Error())
		return
	}

	db := utils.GetCollection("comments")
	if comment.ParentID == nil {
		result, err := db.DeleteMany(context.Background(), bson.M{
			"$or": bson.A{
				bson.M{"_id": id},
				bson.M{"rootId": id},
			},
		})
		if err != nil {
			respondError(c, http.StatusInternalServerError, "楼层评论删除失败: "+err.Error())
			return
		}

		if result.DeletedCount <= 1 {
			respondMessage(c, http.StatusOK, "楼层评论删除成功")
			return
		}

		respondMessage(c, http.StatusOK, fmt.Sprintf("楼层评论删除成功，并删除了 %d 条层内回复", result.DeletedCount-1))
		return
	}

	result, err := db.DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "评论删除失败: "+err.Error())
		return
	}

	if result.DeletedCount == 0 {
		respondError(c, http.StatusNotFound, "找不到要删除的评论")
		return
	}

	respondMessage(c, http.StatusOK, "层内回复删除成功")
}

// LikeComment 给评论点赞
func LikeComment(c *gin.Context) {
	var data struct {
		ID string `json:"_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	id, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的评论ID格式")
		return
	}

	db := utils.GetCollection("comments")
	result, err := db.UpdateByID(context.Background(), id, bson.M{
		"$inc": bson.M{
			"like":      1,
			"likeCount": 1,
		},
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "点赞失败: "+err.Error())
		return
	}

	if result.MatchedCount == 0 {
		respondError(c, http.StatusNotFound, "该评论不存在")
		return
	}

	respondMessage(c, http.StatusOK, "评论点赞成功")
}

// UnLikeComment 取消评论点赞
func UnLikeComment(c *gin.Context) {
	var data struct {
		ID string `json:"_id"`
	}
	if err := c.ShouldBindJSON(&data); err != nil {
		respondError(c, http.StatusBadRequest, "无效的请求数据: "+err.Error())
		return
	}

	id, err := primitive.ObjectIDFromHex(data.ID)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的评论ID格式")
		return
	}

	db := utils.GetCollection("comments")
	filter := bson.M{
		"_id":  id,
		"like": bson.M{"$gt": 0},
	}
	update := bson.M{
		"$inc": bson.M{
			"like":        -1,
			"unlikeCount": 1,
		},
	}

	result, err := db.UpdateOne(context.Background(), filter, update)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "取消点赞失败: "+err.Error())
		return
	}

	if result.MatchedCount == 0 {
		var comment models.Comment
		err := db.FindOne(context.Background(), bson.M{"_id": id}).Decode(&comment)
		if err != nil {
			respondError(c, http.StatusNotFound, "该评论不存在")
		} else {
			respondError(c, http.StatusBadRequest, "点赞数已经是0，无法取消点赞")
		}
		return
	}

	respondMessage(c, http.StatusOK, "评论取消点赞成功")
}

// GetComment 获取某篇博客下的评论楼层列表（分页）
func GetComment(c *gin.Context) {
	db := utils.GetCollection("comments")

	blogStr := c.Param("blogId")
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	blogID, err := primitive.ObjectIDFromHex(blogStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的博客ID格式")
		return
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}
	skip := (page - 1) * limit

	floorFilter := bson.M{
		"blogId":   blogID,
		"parentId": nil,
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: 1}}).
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := db.Find(context.Background(), floorFilter, opts)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "查询评论失败: "+err.Error())
		return
	}
	defer func() {
		if err := cursor.Close(context.Background()); err != nil {
			fmt.Printf("警告: 关闭数据库游标失败: %v\n", err)
		}
	}()

	var floors []models.Comment
	if err = cursor.All(context.Background(), &floors); err != nil {
		respondError(c, http.StatusInternalServerError, "解码评论数据失败: "+err.Error())
		return
	}

	if err := cursor.Err(); err != nil {
		respondError(c, http.StatusInternalServerError, "游标错误: "+err.Error())
		return
	}

	floorCount, err := db.CountDocuments(context.Background(), floorFilter)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "查询楼层总数失败: "+err.Error())
		return
	}

	totalComments, err := db.CountDocuments(context.Background(), bson.M{"blogId": blogID})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "查询评论总数失败: "+err.Error())
		return
	}

	if floors == nil {
		floors = []models.Comment{}
	}

	if len(floors) == 0 {
		respondList(c, []floorCommentResponse{}, &pagination{
			Page:  page,
			Limit: limit,
			Total: floorCount,
		}, gin.H{
			"floorCount":    floorCount,
			"replyCount":    totalComments - floorCount,
			"totalComments": totalComments,
		})
		return
	}

	floorIDs := make([]primitive.ObjectID, 0, len(floors))
	for _, floor := range floors {
		floorIDs = append(floorIDs, floor.ID)
	}

	replyCursor, err := db.Find(
		context.Background(),
		bson.M{"rootId": bson.M{"$in": floorIDs}},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}),
	)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "查询层内回复失败: "+err.Error())
		return
	}
	defer func() {
		if err := replyCursor.Close(context.Background()); err != nil {
			fmt.Printf("警告: 关闭数据库游标失败: %v\n", err)
		}
	}()

	var replies []models.Comment
	if err = replyCursor.All(context.Background(), &replies); err != nil {
		respondError(c, http.StatusInternalServerError, "解码层内回复失败: "+err.Error())
		return
	}

	usernamesByID := make(map[string]string, len(floors)+len(replies))
	for _, floor := range floors {
		usernamesByID[floor.ID.Hex()] = floor.Username
	}
	for _, reply := range replies {
		usernamesByID[reply.ID.Hex()] = reply.Username
	}

	repliesByFloor := make(map[string][]commentResponse, len(floors))
	for _, reply := range replies {
		floorID, ok := commentRootHex(reply)
		if !ok {
			continue
		}

		replyToUsername := reply.ReplyToUsername
		if replyToUsername == "" && reply.ParentID != nil {
			replyToUsername = usernamesByID[reply.ParentID.Hex()]
		}

		repliesByFloor[floorID] = append(repliesByFloor[floorID], buildCommentResponse(reply, false, replyToUsername))
	}

	result := make([]floorCommentResponse, 0, len(floors))
	for _, floor := range floors {
		floorResponse := floorCommentResponse{
			commentResponse: buildCommentResponse(floor, true, ""),
			Replies:         repliesByFloor[floor.ID.Hex()],
		}
		if floorResponse.Replies == nil {
			floorResponse.Replies = []commentResponse{}
		}
		floorResponse.ReplyCount = len(floorResponse.Replies)
		result = append(result, floorResponse)
	}

	respondList(c, result, &pagination{
		Page:  page,
		Limit: limit,
		Total: floorCount,
	}, gin.H{
		"floorCount":    floorCount,
		"replyCount":    totalComments - floorCount,
		"totalComments": totalComments,
	})
}

// GetCommentCount 获取某篇博客的评论总数
func GetCommentCount(c *gin.Context) {
	db := utils.GetCollection("comments")
	blogStr := c.Param("blogId")

	blogID, err := primitive.ObjectIDFromHex(blogStr)
	if err != nil {
		respondError(c, http.StatusBadRequest, "无效的博客ID格式")
		return
	}

	totalComments, err := db.CountDocuments(context.Background(), bson.M{"blogId": blogID})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "搜索评论总数出现错误: "+err.Error())
		return
	}

	floorCount, err := db.CountDocuments(context.Background(), bson.M{
		"blogId":   blogID,
		"parentId": nil,
	})
	if err != nil {
		respondError(c, http.StatusInternalServerError, "搜索楼层评论总数出现错误: "+err.Error())
		return
	}

	respondData(c, http.StatusOK, gin.H{
		"count":      totalComments,
		"floorCount": floorCount,
		"replyCount": totalComments - floorCount,
	})
}

func findCommentByID(id primitive.ObjectID) (models.Comment, error) {
	var comment models.Comment
	err := utils.GetCollection("comments").FindOne(context.Background(), bson.M{"_id": id}).Decode(&comment)
	return comment, err
}

func resolveRootCommentID(parent models.Comment) primitive.ObjectID {
	if parent.ParentID == nil {
		return parent.ID
	}
	if parent.RootID != nil {
		return *parent.RootID
	}
	return *parent.ParentID
}

func buildReplyPrefix(username string) string {
	if username == "" {
		return ""
	}
	return fmt.Sprintf("回复 @%s：", username)
}

func buildDisplayContent(prefix, content string) string {
	if prefix == "" {
		return content
	}
	return prefix + content
}

func buildCommentResponse(comment models.Comment, isFloorOwner bool, replyToUsername string) commentResponse {
	replyPrefix := ""
	if !isFloorOwner {
		replyPrefix = buildReplyPrefix(replyToUsername)
	}

	return commentResponse{
		ID:              comment.ID.Hex(),
		BlogID:          comment.BlogID.Hex(),
		ParentID:        optionalObjectIDHex(comment.ParentID),
		RootID:          optionalObjectIDHex(comment.RootID),
		ReplyToUsername: replyToUsername,
		ReplyPrefix:     replyPrefix,
		Username:        comment.Username,
		QQ:              comment.QQ,
		Content:         comment.Content,
		DisplayContent:  buildDisplayContent(replyPrefix, comment.Content),
		CreatedAt:       comment.CreatedAt,
		Like:            comment.Like,
		LikeCount:       comment.LikeCount,
		UnLikeCount:     comment.UnLikeCount,
		IsFloorOwner:    isFloorOwner,
	}
}

func optionalObjectIDHex(id *primitive.ObjectID) string {
	if id == nil {
		return ""
	}
	return id.Hex()
}

func commentRootHex(comment models.Comment) (string, bool) {
	if comment.RootID != nil {
		return comment.RootID.Hex(), true
	}
	if comment.ParentID != nil {
		return comment.ParentID.Hex(), true
	}
	return "", false
}
