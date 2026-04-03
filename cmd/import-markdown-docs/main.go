package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	defaultMongoURI         = "mongodb://codex:codexcodex@localhost:27017/?authSource=admin"
	defaultDatabaseName     = "asily_blog"
	defaultDocsDir          = "docs/Tanya_xiaomai"
	defaultBlogsCollection  = "blogs"
	defaultImagesCollection = "blog_images"
	defaultTagsCollection   = "tags"
)

var imageRefPattern = regexp.MustCompile(`!\[[^\]]*]\(([^)]+)\)`)

type markdownMeta struct {
	UpdatedAt     string `json:"updated_at"`
	UpdatedAtAlt  string `json:"updatedAt"`
	CreatedAt     string `json:"created_at"`
	CreatedAtAlt  string `json:"createdAt"`
	ReadCount     int    `json:"read_count"`
	ReadCountAlt  int    `json:"readCount"`
	LikeCount     int    `json:"like_count"`
	LikeCountAlt  int    `json:"likeCount"`
}

type importPost struct {
	Title      string
	SourcePath string
	Tags       []string
	Content    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Views      int
	Like       int
	LikeCount  int
	ImageRefs  []string
	ImagePaths []string
}

type importImage struct {
	SourcePath      string
	Filename        string
	BlogSourcePath  string
	BlogTitle       string
	ContentType     string
	Size            int
	Data            []byte
	OriginalRefPath string
}

type importSummary struct {
	ImportedPosts  int
	ImportedImages int
	SkippedFiles   []string
}

func main() {
	var (
		mongoURI         = flag.String("mongo-uri", defaultMongoURI, "MongoDB connection URI")
		dbName           = flag.String("db", defaultDatabaseName, "MongoDB database name")
		docsDir          = flag.String("docs-dir", defaultDocsDir, "Directory containing markdown files")
		blogsCollection  = flag.String("blogs-collection", defaultBlogsCollection, "Blog collection name")
		imagesCollection = flag.String("images-collection", defaultImagesCollection, "Image collection name")
		tagsCollection   = flag.String("tags-collection", defaultTagsCollection, "Tag collection name")
		dryRun           = flag.Bool("dry-run", false, "Parse files and report the result without writing to MongoDB")
	)
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	posts, images, skipped, err := collectImportData(*docsDir)
	if err != nil {
		log.Fatalf("collect import data: %v", err)
	}

	if *dryRun {
		printSummary(importSummary{
			ImportedPosts:  len(posts),
			ImportedImages: len(images),
			SkippedFiles:   skipped,
		})
		return
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(*mongoURI))
	if err != nil {
		log.Fatalf("connect to mongo: %v", err)
	}
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	if err = client.Ping(ctx, nil); err != nil {
		log.Fatalf("ping mongo: %v", err)
	}

	db := client.Database(*dbName)
	if err = upsertPosts(ctx, db.Collection(*blogsCollection), posts); err != nil {
		log.Fatalf("upsert posts: %v", err)
	}
	if err = upsertImages(ctx, db.Collection(*imagesCollection), images); err != nil {
		log.Fatalf("upsert images: %v", err)
	}
	if err = rebuildTags(ctx, db.Collection(*blogsCollection), db.Collection(*tagsCollection)); err != nil {
		log.Fatalf("rebuild tags: %v", err)
	}

	printSummary(importSummary{
		ImportedPosts:  len(posts),
		ImportedImages: len(images),
		SkippedFiles:   skipped,
	})
}

func collectImportData(docsDir string) ([]importPost, []importImage, []string, error) {
	files, err := filepath.Glob(filepath.Join(docsDir, "*.md"))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("glob markdown files: %w", err)
	}

	sort.Strings(files)

	var (
		posts       []importPost
		images      []importImage
		skipped     []string
		seenImageBy = make(map[string]struct{})
	)

	for _, file := range files {
		name := filepath.Base(file)
		if strings.EqualFold(name, "tags.md") {
			skipped = append(skipped, name+" (invalid metadata file)")
			continue
		}

		post, err := parseMarkdownFile(docsDir, file)
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("%s (%v)", name, err))
			continue
		}

		postImages, err := collectImagesForPost(docsDir, file, post)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("collect images for %s: %w", name, err)
		}
		post.ImagePaths = imagePathsFromImages(postImages)

		posts = append(posts, post)
		for _, image := range postImages {
			if _, ok := seenImageBy[image.SourcePath]; ok {
				continue
			}
			seenImageBy[image.SourcePath] = struct{}{}
			images = append(images, image)
		}
	}

	return posts, images, skipped, nil
}

func parseMarkdownFile(docsDir, file string) (importPost, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return importPost{}, fmt.Errorf("read file: %w", err)
	}

	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	lines := strings.Split(content, "\n")
	if len(lines) < 3 {
		return importPost{}, errors.New("file must have at least 3 lines")
	}

	tags, err := parseTagsLine(lines[0])
	if err != nil {
		return importPost{}, fmt.Errorf("parse tags line: %w", err)
	}

	meta, err := parseMetaLine(lines[1])
	if err != nil {
		return importPost{}, fmt.Errorf("parse meta line: %w", err)
	}

	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return importPost{}, fmt.Errorf("load timezone: %w", err)
	}

	createdAt, err := time.ParseInLocation("2006-01-02 15:04:05", meta.CreatedAt, location)
	if err != nil {
		return importPost{}, fmt.Errorf("parse created_at: %w", err)
	}

	updatedAtText := meta.UpdatedAt
	if updatedAtText == "" {
		updatedAtText = meta.CreatedAt
	}

	updatedAt, err := time.ParseInLocation("2006-01-02 15:04:05", updatedAtText, location)
	if err != nil {
		return importPost{}, fmt.Errorf("parse updated_at: %w", err)
	}

	body := strings.Join(lines[2:], "\n")
	title := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	sourcePath, err := filepath.Rel(docsDir, file)
	if err != nil {
		return importPost{}, fmt.Errorf("build source path: %w", err)
	}

	imageRefs := findLocalImageRefs(body)

	return importPost{
		Title:      title,
		SourcePath: filepath.ToSlash(sourcePath),
		Tags:       tags,
		Content:    body,
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
		Views:      meta.ReadCount,
		Like:       meta.LikeCount,
		LikeCount:  meta.LikeCount,
		ImageRefs:  imageRefs,
	}, nil
}

func parseTagsLine(line string) ([]string, error) {
	var tags []string
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &tags); err != nil {
		return nil, err
	}
	return tags, nil
}

func parseMetaLine(line string) (markdownMeta, error) {
	var meta markdownMeta
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &meta); err != nil {
		return markdownMeta{}, err
	}

	if meta.CreatedAt == "" {
		meta.CreatedAt = meta.CreatedAtAlt
	}
	if meta.UpdatedAt == "" {
		meta.UpdatedAt = meta.UpdatedAtAlt
	}
	if meta.ReadCount == 0 && meta.ReadCountAlt != 0 {
		meta.ReadCount = meta.ReadCountAlt
	}
	if meta.LikeCount == 0 && meta.LikeCountAlt != 0 {
		meta.LikeCount = meta.LikeCountAlt
	}

	if meta.CreatedAt == "" {
		return markdownMeta{}, errors.New("created_at is required")
	}
	return meta, nil
}

func findLocalImageRefs(body string) []string {
	matches := imageRefPattern.FindAllStringSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var refs []string
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		ref := strings.TrimSpace(match[1])
		ref = strings.Trim(ref, `"'`)
		if ref == "" || strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") || strings.HasPrefix(ref, "data:") {
			continue
		}

		ref = filepath.ToSlash(ref)
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}

	sort.Strings(refs)
	return refs
}

func collectImagesForPost(docsDir, file string, post importPost) ([]importImage, error) {
	if len(post.ImageRefs) == 0 {
		return nil, nil
	}

	baseDir := filepath.Dir(file)
	var images []importImage
	for _, ref := range post.ImageRefs {
		absolutePath := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(ref)))
		data, err := os.ReadFile(absolutePath)
		if err != nil {
			return nil, fmt.Errorf("read image %s: %w", absolutePath, err)
		}

		sourcePath, err := filepath.Rel(docsDir, absolutePath)
		if err != nil {
			return nil, fmt.Errorf("build image source path: %w", err)
		}

		ext := strings.ToLower(filepath.Ext(absolutePath))
		contentType := mime.TypeByExtension(ext)
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		images = append(images, importImage{
			SourcePath:      filepath.ToSlash(sourcePath),
			Filename:        filepath.Base(absolutePath),
			BlogSourcePath:  post.SourcePath,
			BlogTitle:       post.Title,
			ContentType:     contentType,
			Size:            len(data),
			Data:            data,
			OriginalRefPath: ref,
		})
	}

	return images, nil
}

func imagePathsFromImages(images []importImage) []string {
	if len(images) == 0 {
		return nil
	}

	paths := make([]string, 0, len(images))
	for _, image := range images {
		paths = append(paths, image.SourcePath)
	}

	sort.Strings(paths)
	return paths
}

func upsertPosts(ctx context.Context, collection *mongo.Collection, posts []importPost) error {
	for _, post := range posts {
		imageRefs := post.ImageRefs
		if imageRefs == nil {
			imageRefs = []string{}
		}

		imagePaths := post.ImagePaths
		if imagePaths == nil {
			imagePaths = []string{}
		}

		update := bson.M{
			"$set": bson.M{
				"title":       post.Title,
				"sourcePath":  post.SourcePath,
				"content":     post.Content,
				"tags":        post.Tags,
				"createdAt":   post.CreatedAt,
				"updatedAt":   post.UpdatedAt,
				"views":       post.Views,
				"like":        post.Like,
				"likeCount":   post.LikeCount,
				"unlikeCount": 0,
				"imageRefs":   imageRefs,
				"imagePaths":  imagePaths,
				"importedAt":  time.Now(),
				"contentType": "markdown",
			},
		}

		_, err := collection.UpdateOne(
			ctx,
			bson.M{"sourcePath": post.SourcePath},
			update,
			options.Update().SetUpsert(true),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func upsertImages(ctx context.Context, collection *mongo.Collection, images []importImage) error {
	for _, image := range images {
		update := bson.M{
			"$set": bson.M{
				"sourcePath":      image.SourcePath,
				"filename":        image.Filename,
				"blogSourcePath":  image.BlogSourcePath,
				"blogTitle":       image.BlogTitle,
				"contentType":     image.ContentType,
				"size":            image.Size,
				"data":            image.Data,
				"originalRefPath": image.OriginalRefPath,
				"importedAt":      time.Now(),
			},
		}

		_, err := collection.UpdateOne(
			ctx,
			bson.M{"sourcePath": image.SourcePath},
			update,
			options.Update().SetUpsert(true),
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func rebuildTags(ctx context.Context, blogsCollection, tagsCollection *mongo.Collection) error {
	cursor, err := blogsCollection.Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"tags": bson.M{"$exists": true, "$ne": bson.A{}}}}},
		{{Key: "$unwind", Value: "$tags"}},
		{{Key: "$group", Value: bson.M{"_id": "$tags", "count": bson.M{"$sum": 1}}}},
		{{Key: "$sort", Value: bson.M{"count": -1, "_id": 1}}},
	})
	if err != nil {
		return err
	}
	defer func() {
		_ = cursor.Close(context.Background())
	}()

	var rows []struct {
		Name  string `bson:"_id"`
		Count int    `bson:"count"`
	}
	if err = cursor.All(ctx, &rows); err != nil {
		return err
	}

	if _, err = tagsCollection.DeleteMany(ctx, bson.D{}); err != nil {
		return err
	}

	if len(rows) == 0 {
		return nil
	}

	docs := make([]interface{}, 0, len(rows))
	for _, row := range rows {
		docs = append(docs, bson.M{
			"name":  row.Name,
			"count": row.Count,
		})
	}

	_, err = tagsCollection.InsertMany(ctx, docs)
	return err
}

func printSummary(summary importSummary) {
	fmt.Printf("Imported posts: %d\n", summary.ImportedPosts)
	fmt.Printf("Imported images: %d\n", summary.ImportedImages)
	if len(summary.SkippedFiles) == 0 {
		fmt.Println("Skipped files: 0")
		return
	}

	fmt.Printf("Skipped files: %d\n", len(summary.SkippedFiles))
	for _, item := range summary.SkippedFiles {
		fmt.Printf("  - %s\n", item)
	}
}
