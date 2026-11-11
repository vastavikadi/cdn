type FileCategory = "image" | "scripts" | "style" | "documents" | "unknown";

function getFileCategory(file: File): FileCategory {
  switch (file.type) {
    case "image/jpeg":
    case "image/png":
    case "image/gif":
      return "image";

    case "application/javascript":
    case "text/javascript":
      return "scripts";

    case "text/css":
      return "style";

    case "application/pdf":
    case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
      return "documents";

    default:
      return "unknown";
  }
}

export default getFileCategory;
