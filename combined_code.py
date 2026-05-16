# combined_code.py
from pathlib import Path
from typing import Iterable, List, Optional, Set

# ================== НАСТРОЙКИ ==================

OUTPUT_FILE = "combined_modules.txt"

INCLUDE_EXTS: Set[str] = {".py", ".ts", ".js", ".vue", ".go", ".mod", ".proto"}

EXCLUDE_DIRS: Set[str] = {
    ".venv", "venv", ".git", ".idea", "__pycache__", "alembic",
    ".mypy_cache", ".pytest_cache", ".ruff_cache", ".tox",
    "dist", "build", "node_modules", ".output", ".nuxt",
    ".next", ".vite", ".turbo", "coverage",
}

# файлы в корне проекта, которые нужно исключить
EXCLUDE_FILES: Set[str] = {
    "combined_code.py",
    "chank_combined_code.py",
    "main.py",  # если это точка входа, а не часть проекта
}


# ===============================================


def to_rel_posix(path: Path, root: Path) -> str:
    return path.relative_to(root).as_posix()


def iter_files(root: Path, start: Path) -> List[Path]:
    files: List[Path] = []

    if start.is_file():
        return [start] if start.suffix in INCLUDE_EXTS else []

    for p in start.rglob("*"):
        if not p.is_file():
            continue
        if p.suffix not in INCLUDE_EXTS:
            continue
        try:
            rel_parts = p.relative_to(root).parts
        except ValueError:
            continue
        if any(part in EXCLUDE_DIRS for part in rel_parts[:-1]):
            continue
        files.append(p)

    return sorted(files, key=lambda x: x.as_posix())


def header_for(file_path: Path, root: Path) -> str:
    rel = to_rel_posix(file_path, root)
    if file_path.suffix == ".vue":
        return f"<!--{rel}-->\n"
    if file_path.suffix in {".ts", ".js"}:
        return f"// {rel}\n"
    return f"# {rel}\n"


def is_any_supported_header_line(line: str) -> bool:
    s = line.strip()
    if s.startswith("# ") and s.endswith((".py", ".ts", ".js", ".vue")):
        return True
    if s.startswith("// ") and s.endswith((".py", ".ts", ".js", ".vue")):
        return True
    if s.startswith("<!--") and s.endswith("-->"):
        inner = s.removeprefix("<!--").removesuffix("-->").strip()
        if inner.endswith((".py", ".ts", ".js", ".vue")):
            return True
    return False


def ensure_header(file_path: Path, root: Path) -> bool:
    expected = header_for(file_path, root)
    text = file_path.read_text(encoding="utf-8", errors="replace")
    lines = text.splitlines(keepends=True)

    if not lines:
        file_path.write_text(expected, encoding="utf-8")
        return True
    if lines[0] == expected:
        return False
    if is_any_supported_header_line(lines[0]):
        lines[0] = expected
    else:
        lines.insert(0, expected)

    file_path.write_text("".join(lines), encoding="utf-8")
    return True


def build_tree(files: List[Path], root: Path) -> str:
    """Строит текстовую схему структуры папок и файлов."""
    from collections import defaultdict

    # Собираем все уникальные директории и файлы
    # Структура: { parent_tuple: { "dirs": set, "files": list } }
    tree: dict = defaultdict(lambda: {"dirs": set(), "files": []})

    for p in files:
        parts = p.relative_to(root).parts  # например: ("app", "api", "auth.py")

        # Регистрируем файл у его родителя
        parent = parts[:-1]  # ("app", "api")
        tree[parent]["files"].append(parts[-1])

        # Регистрируем каждую директорию у её родителя
        for depth in range(len(parts) - 1):
            dir_parent = parts[:depth]  # ()  затем  ("app",)
            dir_name = parts[depth]  # "app"  затем  "api"
            tree[dir_parent]["dirs"].add(dir_name)

    lines: List[str] = ["./"]

    def render(node_key: tuple, prefix: str = "") -> None:
        node = tree[node_key]
        dirs = sorted(node["dirs"])
        fls = sorted(node["files"])
        items = [("dir", d) for d in dirs] + [("file", f) for f in fls]

        for i, (kind, name) in enumerate(items):
            is_last = i == len(items) - 1
            connector = "└── " if is_last else "├── "
            child_pfx = prefix + ("    " if is_last else "│   ")

            lines.append(f"{prefix}{connector}{name}{'/' if kind == 'dir' else ''}")
            if kind == "dir":
                render(node_key + (name,), child_pfx)

    render(())
    return "\n".join(lines)


def build_combined(files: List[Path], root: Path, out_file: Path) -> None:
    tree_str = build_tree(files, root)

    header_section = (
            "Вот структура проекта:\n\n"
            + tree_str
            + "\n\n"
            + "Можешь искать нужные файлы в проекте в этом документе, схема выше.\n"
    )

    separator = "\n======================\n\n"

    blocks: List[str] = [header_section]
    for p in files:
        blocks.append(p.read_text(encoding="utf-8", errors="replace").rstrip() + "\n")

    out_file.write_text(
        separator.join(blocks).rstrip() + "\n",
        encoding="utf-8",
    )


def main(path: Optional[str] = None) -> None:
    root = Path(__file__).resolve().parent
    script_path = Path(__file__).resolve()
    out_file = root / OUTPUT_FILE

    start = (root / path).resolve() if path else root

    files = iter_files(root, start)

    files = [
        p for p in files
        if p.resolve() not in {script_path, out_file.resolve()}
           and p.name not in EXCLUDE_FILES
    ]

    changed = 0
    for p in files:
        if ensure_header(p, root):
            changed += 1

    build_combined(files, root, out_file)

    print(f"Files processed: {len(files)}")
    print(f"Headers updated: {changed}")
    print(f"Combined file:   {out_file.name}")


if __name__ == "__main__":
    main()
