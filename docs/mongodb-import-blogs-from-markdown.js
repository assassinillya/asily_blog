// Import markdown blogs from docs/Tanya_xiaomai into MongoDB.
// Run this script from the repository root:
//   mongosh "mongodb://<user>:<pass>@localhost:27017/?authSource=admin" docs/mongodb-import-blogs-from-markdown.js

db = db.getSiblingDB("asily_blog");

const fs = require("fs");
const path = require("path");

const repoRoot = process.cwd();
const docsDir = path.join(repoRoot, "docs", "Tanya_xiaomai");
const imagesDir = path.join(docsDir, "images");

const skipFiles = new Set(["tags.md"]);
const imageRefPattern = /!\[[^\]]*]\(([^)]+)\)/g;

function stripBom(text) {
  return text.replace(/^\uFEFF/, "");
}

function readMarkdownFiles(dir) {
  return fs
    .readdirSync(dir, { withFileTypes: true })
    .filter((entry) => entry.isFile() && entry.name.toLowerCase().endsWith(".md"))
    .map((entry) => entry.name)
    .sort((a, b) => a.localeCompare(b, "zh-CN"));
}

function parseMarkdownFile(fileName) {
  const fullPath = path.join(docsDir, fileName);
  const raw = stripBom(fs.readFileSync(fullPath, "utf8")).replace(/\r\n/g, "\n");
  const lines = raw.split("\n");

  if (lines.length < 3) {
    throw new Error("file must have at least 3 lines");
  }

  const tags = JSON.parse(lines[0]);
  const meta = JSON.parse(lines[1]);
  const content = lines.slice(2).join("\n");

  if (!Array.isArray(tags)) {
    throw new Error("first line must be a JSON array");
  }
  if (!meta.created_at) {
    throw new Error("second line must contain created_at");
  }

  const createdAt = new Date(meta.created_at.replace(" ", "T") + "+08:00");
  if (Number.isNaN(createdAt.getTime())) {
    throw new Error("invalid created_at format");
  }

  const imageRefs = extractImageRefs(content);

  return {
    title: path.basename(fileName, ".md"),
    sourcePath: fileName.replace(/\\/g, "/"),
    tags,
    meta,
    content,
    createdAt,
    updatedAt: createdAt,
    imageRefs
  };
}

function extractImageRefs(content) {
  const refs = new Set();
  let match;
  while ((match = imageRefPattern.exec(content)) !== null) {
    const ref = (match[1] || "").trim().replace(/^["']|["']$/g, "");
    if (!ref) {
      continue;
    }
    if (/^(https?:)?\/\//i.test(ref) || /^data:/i.test(ref)) {
      continue;
    }
    refs.add(ref.replace(/\\/g, "/"));
  }
  return Array.from(refs).sort();
}

function guessContentType(filePath) {
  const ext = path.extname(filePath).toLowerCase();
  switch (ext) {
    case ".png":
      return "image/png";
    case ".jpg":
    case ".jpeg":
      return "image/jpeg";
    case ".gif":
      return "image/gif";
    case ".webp":
      return "image/webp";
    case ".svg":
      return "image/svg+xml";
    default:
      return "application/octet-stream";
  }
}

function importImage(blogDoc, imageRef) {
  const absoluteImagePath = path.resolve(docsDir, blogDoc.sourcePath, "..", imageRef);
  if (!fs.existsSync(absoluteImagePath)) {
    throw new Error("image not found: " + absoluteImagePath);
  }

  const relativeSourcePath = path.relative(docsDir, absoluteImagePath).replace(/\\/g, "/");
  const bytes = fs.readFileSync(absoluteImagePath);

  db.blog_images.updateOne(
    { sourcePath: relativeSourcePath },
    {
      $set: {
        sourcePath: relativeSourcePath,
        filename: path.basename(absoluteImagePath),
        blogSourcePath: blogDoc.sourcePath,
        blogTitle: blogDoc.title,
        originalRefPath: imageRef,
        contentType: guessContentType(absoluteImagePath),
        size: bytes.length,
        data: new Binary(bytes, 0),
        importedAt: new Date()
      }
    },
    { upsert: true }
  );
}

function importBlogDoc(blogDoc) {
  const existing = db.blogs.findOne(
    { sourcePath: blogDoc.sourcePath },
    { projection: { like: 1, likeCount: 1, unlikeCount: 1 } }
  );

  const like = existing && typeof existing.like === "number" ? existing.like : Number(blogDoc.meta.like_count || 0);
  const likeCount =
    existing && typeof existing.likeCount === "number" ? existing.likeCount : Number(blogDoc.meta.like_count || 0);
  const unlikeCount =
    existing && typeof existing.unlikeCount === "number" ? existing.unlikeCount : 0;

  db.blogs.updateOne(
    { sourcePath: blogDoc.sourcePath },
    {
      $set: {
        title: blogDoc.title,
        sourcePath: blogDoc.sourcePath,
        content: blogDoc.content,
        tags: blogDoc.tags,
        createdAt: blogDoc.createdAt,
        updatedAt: blogDoc.updatedAt,
        views: Number(blogDoc.meta.read_count || 0),
        like,
        likeCount,
        unlikeCount,
        imageRefs: blogDoc.imageRefs,
        contentType: "markdown",
        importedAt: new Date()
      }
    },
    { upsert: true }
  );
}

function rebuildTags() {
  const rows = db.blogs
    .aggregate([
      { $match: { tags: { $exists: true, $ne: [] } } },
      { $unwind: "$tags" },
      { $group: { _id: "$tags", count: { $sum: 1 } } },
      { $sort: { count: -1, _id: 1 } }
    ])
    .toArray();

  db.tags.deleteMany({});
  if (rows.length > 0) {
    db.tags.insertMany(
      rows.map((row) => ({
        name: row._id,
        count: row.count
      }))
    );
  }
}

function ensureImageIndexes() {
  db.blog_images.createIndex({ sourcePath: 1 }, { name: "idx_sourcePath", unique: true });
  db.blog_images.createIndex({ blogSourcePath: 1 }, { name: "idx_blogSourcePath" });
}

const result = {
  importedBlogs: 0,
  importedImages: 0,
  skippedFiles: [],
  failedFiles: []
};

ensureImageIndexes();

for (const fileName of readMarkdownFiles(docsDir)) {
  if (skipFiles.has(fileName)) {
    result.skippedFiles.push(fileName);
    continue;
  }

  try {
    const blogDoc = parseMarkdownFile(fileName);
    importBlogDoc(blogDoc);
    result.importedBlogs += 1;

    for (const imageRef of blogDoc.imageRefs) {
      importImage(blogDoc, imageRef);
      result.importedImages += 1;
    }
  } catch (error) {
    result.failedFiles.push({
      file: fileName,
      error: String(error.message || error)
    });
  }
}

rebuildTags();

printjson(result);
