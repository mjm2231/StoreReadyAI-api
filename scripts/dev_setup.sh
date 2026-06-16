#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT}/scripts/versions.env"

DEV="${ROOT}/.dev"
BIN="${DEV}/bin"
ETC="${DEV}/etc"
DATA="${ROOT}/.data"
DIST="${DEV}/dist"

mkdir -p "${BIN}" "${ETC}" "${DATA}"/{mysql,redis,logs} "${DIST}"

OS="$(uname -s)"
ARCH="$(uname -m)"
[[ "${OS}" == "Darwin" ]] || { echo "macOS only"; exit 1; }

# ---------- utils ----------
log() { echo -e "==> $*"; }

sha256_check() {
  local file="$1" expected="$2"
  [[ -n "${expected}" ]] || return 0
  local got
  got="$(shasum -a 256 "${file}" | awk '{print $1}')"
  [[ "${got}" == "${expected}" ]] || { echo "SHA256 mismatch: ${file}"; exit 1; }
}

curl_dl() {
  local url="$1" out="$2"
  log "Download: ${url}"

  # 说明：macOS 自带 curl 常见 LibreSSL/HTTP2/IPv6 网络握手问题（如 SSL_ERROR_SYSCALL）。
  # 这里做多策略重试，优先保证下载成功。
  local base_opts=(
    -fL
    --retry 8
    --retry-delay 1
    --retry-all-errors
    --connect-timeout 15
    --max-time 0
    -o "${out}"
  )

  # 允许用户额外覆盖 curl 参数（例如走代理）
  # 在 versions.env 里可设置：CURL_EXTRA_OPTS="--proxy http://127.0.0.1:7890"
  local extra=()
  if [[ -n "${CURL_EXTRA_OPTS:-}" ]]; then
    # shellcheck disable=SC2206
    extra=(${CURL_EXTRA_OPTS})
  fi

  if curl "${base_opts[@]}" "${extra[@]}" "${url}"; then
    return 0
  fi

  log "curl failed, retry with --http1.1 --tlsv1.2"
  if curl "${base_opts[@]}" --http1.1 --tlsv1.2 "${extra[@]}" "${url}"; then
    return 0
  fi

  log "curl failed, retry with IPv4 (-4)"
  if curl "${base_opts[@]}" -4 --http1.1 --tlsv1.2 "${extra[@]}" "${url}"; then
    return 0
  fi

  echo "Download failed after retries: ${url}" >&2
  return 1
}

unquarantine() {
  local p="$1"
  # 避免 macOS Gatekeeper 隔离位导致无法执行
  xattr -dr com.apple.quarantine "${p}" >/dev/null 2>&1 || true
}

# 如果同一个 dmg 之前已经 attach 但未正常 detach，会导致再次 attach 报“资源忙”。
# 这里尝试根据 hdiutil info 找到对应的 /dev/diskX 并强制 detach。
detach_dmg_if_attached() {
  local img="$1"
  [[ -n "${img}" ]] || return 0

  # hdiutil info 的输出会包含形如：
  # /dev/disk4           GUID_partition_scheme
  #   ...
  #   image-path: /path/to/xxx.dmg
  # 我们记录最近一次出现的 /dev/diskX，然后在匹配到 image-path 时输出该 disk。
  local devs
  devs="$(hdiutil info 2>/dev/null | awk -v img="${img}" '
    /^\/dev\/disk[0-9]+/ { dev=$1 }
    $0 ~ /^\s*image-path:/ {
      if (index($0, img) > 0 && dev != "") print dev
    }
  ' | sort -u)"

  if [[ -n "${devs}" ]]; then
    while IFS= read -r d; do
      [[ -n "${d}" ]] || continue
      log "Detach existing MySQL dmg device: ${d}"
      hdiutil detach "${d}" -force -quiet >/dev/null 2>&1 || true
    done <<< "${devs}"
  fi
}

# ---------- Redis: source build (no official macOS binary) ----------
install_redis() {
  local prefix="${BIN}/redis"
  if [[ -x "${prefix}/redis-server" ]]; then
    log "Redis already installed: ${prefix}"
    return
  fi

  mkdir -p "${prefix}" "${DIST}"
  local tgz="${DIST}/redis-${REDIS_VER}.tar.gz"

  # Redis release tarball：从 GitHub releases 获取源码包（稳定）  [oai_citation:5‡GitHub](https://github.com/redis/redis/releases?utm_source=chatgpt.com)
  local url="https://github.com/redis/redis/archive/refs/tags/${REDIS_VER}.tar.gz"
  curl_dl "${url}" "${tgz}"

  log "Build Redis ${REDIS_VER}"
  rm -rf "${DIST}/redis-src"
  mkdir -p "${DIST}/redis-src"
  tar -xzf "${tgz}" -C "${DIST}/redis-src" --strip-components=1

  (cd "${DIST}/redis-src" && make -j"$(sysctl -n hw.ncpu)")

  cp -f "${DIST}/redis-src/src/redis-server" "${prefix}/redis-server"
  cp -f "${DIST}/redis-src/src/redis-cli"    "${prefix}/redis-cli"
  chmod +x "${prefix}/redis-server" "${prefix}/redis-cli"
  unquarantine "${prefix}"
}

# ---------- MinIO: official darwin binaries directory ----------
install_minio() {
  local prefix="${BIN}/minio"
  if [[ -x "${prefix}/minio" ]]; then
    log "MinIO already installed: ${prefix}"
    return
  fi
  mkdir -p "${prefix}"
  local plat="darwin-amd64"
  [[ "${ARCH}" == "arm64" ]] && plat="darwin-arm64"

  # 官方目录：dl.min.io/server/minio/release/ 有 darwin-arm64/amd64  [oai_citation:6‡dl.min.io](https://dl.min.io/server/minio/release/?utm_source=chatgpt.com)
  local url="https://dl.min.io/server/minio/release/${plat}/minio"
  curl_dl "${url}" "${prefix}/minio"
  chmod +x "${prefix}/minio"
  unquarantine "${prefix}"
}

# ---------- JDK (for Kafka): use Oracle JDK tar.gz (stable direct link) ----------
install_jdk() {
  local prefix="${BIN}/jdk"
  if [[ -x "${prefix}/bin/java" ]]; then
    log "JDK already installed: ${prefix}"
    return
  fi
  mkdir -p "${prefix}" "${DIST}"

  local arch="x64"
  [[ "${ARCH}" == "arm64" ]] && arch="aarch64"

  # Oracle JDK 21 macOS tar.gz 有稳定直链（含 aarch64）  [oai_citation:7‡Oracle](https://www.oracle.com/asean/java/technologies/downloads/?utm_source=chatgpt.com)
  local url="https://download.oracle.com/java/21/latest/jdk-21_macos-${arch}_bin.tar.gz"
  local tgz="${DIST}/jdk21-macos-${arch}.tar.gz"
  curl_dl "${url}" "${tgz}"

  rm -rf "${prefix}"
  mkdir -p "${prefix}"
  tar -xzf "${tgz}" -C "${prefix}" --strip-components=1
  unquarantine "${prefix}"
}

# ---------- Kafka: binary tgz (needs JDK) ----------
install_kafka() {
  local prefix="${BIN}/kafka"
  if [[ -x "${prefix}/bin/kafka-server-start.sh" ]]; then
    log "Kafka already installed: ${prefix}"
    return
  fi
  mkdir -p "${prefix}" "${DIST}"

  # 官方下载页给出 kafka_2.12-3.7.0.tgz / sha512  [oai_citation:8‡kafka.apache.org](https://kafka.apache.org/community/downloads/?utm_source=chatgpt.com)
  local scala="2.12"
  local tgz_name="kafka_${scala}-${KAFKA_VER}.tgz"
  local url="https://archive.apache.org/dist/kafka/${KAFKA_VER}/${tgz_name}"
  local tgz="${DIST}/${tgz_name}"
  curl_dl "${url}" "${tgz}"

  rm -rf "${prefix}"
  mkdir -p "${prefix}"
  tar -xzf "${tgz}" -C "${prefix}" --strip-components=1
  unquarantine "${prefix}"
}

# ---------- Jaeger: download binary from official download page (via GitHub releases usually) ----------
# 这里给“通用做法”：你也可以直接改成固定 URL
install_jaeger() {
  local prefix="${BIN}/jaeger"
  if [[ -x "${prefix}/jaeger-all-in-one" ]]; then
    log "Jaeger already installed: ${prefix}"
    return
  fi
  mkdir -p "${prefix}" "${DIST}"

  # 官方下载页说明各平台二进制可用  [oai_citation:9‡Jaeger](https://www.jaegertracing.io/download/?utm_source=chatgpt.com)
  # 为了脚本稳定：建议你后续把最终 URL 固化到 versions.env（你升级时改一处）
  echo "Jaeger: please pin the exact asset URL for your version in versions.env (recommended)."
}

# ---------- MySQL: download dmg/pkg and extract into .dev/bin/mysql ----------
# MySQL 在 macOS 上官方以 dmg/pkg 分发  [oai_citation:10‡dev.mysql.com](https://dev.mysql.com/doc/refman/8.4/en/macos-installation-pkg.html?utm_source=chatgpt.com)
# 由于 MySQL 下载页面的直链/参数可能变动，这里采用“可维护的直链变量”：
# 你只需要在 versions.env 写 MYSQL_MAC_URL（团队固定版本即可）。
install_mysql() {
  local prefix="${BIN}/mysql"
  if [[ -x "${prefix}/bin/mysqld" ]]; then
    log "MySQL already installed: ${prefix}"
    return
  fi

  # 允许手动固定直链（团队锁版本时很有用），但默认自动从下载页解析真实文件名，避免 404
  : "${MYSQL_MAC_URL:=}"
  # 可选：优先使用 tar.gz（无需挂载 dmg，速度快且更稳定）
  : "${MYSQL_MAC_TAR_URL:=}"

  if [[ -z "${MYSQL_MAC_URL}" ]]; then
    # 由 MYSQL_VER 计算 series（例如 8.4.8 -> 8.4）
    local series
    series="$(echo "${MYSQL_VER}" | awk -F. '{print $1"."$2}')"

    # MySQL 下载页可按 os 参数切换平台；os=33 是 macOS
    local dl_page="https://dev.mysql.com/downloads/mysql/${series}.html?os=33"

    # 构造文件名匹配：arm64 / x86_64
    local arch_tag="x86_64"
    [[ "${ARCH}" == "arm64" ]] && arch_tag="arm64"

    log "Resolve MySQL DMG from download page: ${dl_page}"

    # 先尝试解析 tar.gz（无需挂载 dmg，避免 hdiutil 偶发“资源忙”）
    local tar_name
    tar_name="$(curl -fsSL "${dl_page}" | grep -Eo "mysql-${MYSQL_VER}-macos[^\"']*-${arch_tag}\.tar\.gz" | head -n 1 || true)"
    if [[ -n "${tar_name}" ]]; then
      MYSQL_MAC_TAR_URL="https://dev.mysql.com/get/Downloads/MySQL-${series}/${tar_name}"
    fi

    # 再解析 dmg 作为兜底
    local dmg_name
    dmg_name="$(curl -fsSL "${dl_page}" | grep -Eo "mysql-${MYSQL_VER}-macos[^\"']*-${arch_tag}\.dmg" | head -n 1 || true)"
    if [[ -n "${dmg_name}" ]]; then
      MYSQL_MAC_URL="https://dev.mysql.com/get/Downloads/MySQL-${series}/${dmg_name}"
    fi

    if [[ -z "${MYSQL_MAC_TAR_URL}" && -z "${MYSQL_MAC_URL}" ]]; then
      cat <<EOF
[MySQL] 无法从下载页解析安装包文件名（tar.gz / dmg 都未找到，可能是版本/页面结构变更）
- 尝试打开并确认该版本是否存在：${dl_page}
- 临时修复：在 scripts/versions.env 固定直链，例如：
  MYSQL_MAC_TAR_URL="https://dev.mysql.com/get/Downloads/MySQL-${series}/mysql-${MYSQL_VER}-macosXX-${arch_tag}.tar.gz"
  MYSQL_MAC_URL="https://dev.mysql.com/get/Downloads/MySQL-${series}/mysql-${MYSQL_VER}-macosXX-${arch_tag}.dmg"
EOF
      exit 1
    fi
  fi

  # 优先使用 tar.gz（更稳定），否则回退 dmg/pkg
  if [[ -n "${MYSQL_MAC_TAR_URL}" ]]; then
    local tgz="${DIST}/${MYSQL_VER}-macos-${ARCH}.tar.gz"
    curl_dl "${MYSQL_MAC_TAR_URL}" "${tgz}"
    log "MySQL TAR URL: ${MYSQL_MAC_TAR_URL}"
    log "MySQL TAR: ${tgz}"

    log "Extract MySQL from tar.gz (no system install)"

    rm -rf "${prefix}"
    mkdir -p "${prefix}"

    # tar.gz 通常包含顶层目录 mysql-<ver>-macosXX-<arch>/
    local tmp="${DIST}/mysql-tar"
    rm -rf "${tmp}"
    mkdir -p "${tmp}"
    tar -xzf "${tgz}" -C "${tmp}"

    local top
    # 注意：tmp 目录名是 mysql-tar，可能会误匹配 mysql-*；因此用 -mindepth 1 只匹配解压出来的子目录。
    top="$(find "${tmp}" -mindepth 1 -maxdepth 1 -type d -name "mysql-*" | head -n 1 || true)"
    [[ -n "${top}" ]] || { echo "[MySQL] tar.gz 解压后未找到 mysql-* 顶层目录"; ls -la "${tmp}"; exit 1; }

    # 兼容两种布局：
    # 1) <top>/bin <top>/lib <top>/share
    # 2) <top>/usr/local/mysql/{bin,lib,share}
    local base="${top}"
    if [[ -d "${top}/usr/local/mysql" ]]; then
      base="${top}/usr/local/mysql"
    fi

    [[ -d "${base}/bin" && -d "${base}/lib" ]] || {
      echo "[MySQL] tar.gz 布局异常，未找到 bin/lib：base=${base}";
      find "${top}" -maxdepth 3 -type d | head -n 80;
      exit 1;
    }

    # 复制 bin/lib/share 到最终目录
    cp -R "${base}/bin" "${prefix}/bin"
    cp -R "${base}/lib" "${prefix}/lib"
    [[ -d "${base}/share" ]] && cp -R "${base}/share" "${prefix}/share" || true

    chmod -R u+rwX,go+rX "${prefix}" || true
    unquarantine "${prefix}"
    return
  fi

  local dmg="${DIST}/${MYSQL_VER}-macos-${ARCH}.dmg"
  curl_dl "${MYSQL_MAC_URL}" "${dmg}"
  log "MySQL URL: ${MYSQL_MAC_URL}"
  log "MySQL DMG: ${dmg}"

  log "Extract MySQL from dmg/pkg (no system install)"

  # mountpoint 放到 /tmp 更干净，避免项目目录下的文件监听/索引/权限导致“资源忙”。
  local mnt="/tmp/storeready_ai-mysql-mnt-$$"

  # 如果之前异常退出导致同名 mountpoint 仍被占用，先强制卸载。
  if mount | grep -q " ${mnt} "; then
    hdiutil detach "${mnt}" -force -quiet >/dev/null 2>&1 || true
    diskutil unmount force "${mnt}" >/dev/null 2>&1 || true
  fi

  rm -rf "${mnt}" >/dev/null 2>&1 || true
  mkdir -p "${mnt}"

  # 确保函数退出时一定卸载，避免残留导致下次运行失败。
  local _mysql_cleanup
  _mysql_cleanup() {
    hdiutil detach "${mnt}" -force -quiet >/dev/null 2>&1 || true
    diskutil unmount force "${mnt}" >/dev/null 2>&1 || true
    rm -rf "${mnt}" >/dev/null 2>&1 || true
  }
  trap _mysql_cleanup RETURN

  # 挂载 DMG：偶发会遇到“资源忙”，通常是之前 attach 残留或系统占用导致。
  # 处理策略：先尝试 detach 同名 dmg 的残留设备，再重试几次；失败时输出详细错误。
  detach_dmg_if_attached "${dmg}"

  local _attach_ok=0
  for i in 1 2 3 4 5; do
    # -readonly：明确只读挂载；-noverify：跳过校验加速并避免偶发阻塞；-noautoopen：不弹窗口
    if hdiutil attach "${dmg}" -mountpoint "${mnt}" -nobrowse -readonly -noverify -noautoopen -quiet; then
      _attach_ok=1
      break
    fi
    # 再次尝试前先 detach 残留并稍等。
    detach_dmg_if_attached "${dmg}"
    sleep 1
  done

  if [[ "${_attach_ok}" -ne 1 ]]; then
    echo "[MySQL] hdiutil attach 失败（多次重试后仍失败），输出详细错误信息..." >&2
    hdiutil attach "${dmg}" -mountpoint "${mnt}" -nobrowse -readonly -noverify -noautoopen || true
    exit 1
  fi

  # 找到 pkg
  local pkg
  pkg="$(find "${mnt}" -maxdepth 2 -name "*.pkg" | head -n 1 || true)"
  [[ -n "${pkg}" ]] || { hdiutil detach "${mnt}" -quiet || true; echo "MySQL pkg not found in dmg"; exit 1; }

  local exp="${DIST}/mysql-pkg"
  rm -rf "${exp}"
  mkdir -p "${exp}"
  if ! pkgutil --expand "${pkg}" "${exp}"; then
    echo "[MySQL] pkgutil --expand 失败：${pkg}" >&2
    exit 1
  fi

  # payload 通常在 *.pkg/Payload
  local payload
  payload="$(find "${exp}" -name "Payload" | head -n 1 || true)"
  [[ -n "${payload}" ]] || { hdiutil detach "${mnt}" -quiet || true; echo "MySQL Payload not found"; exit 1; }

  rm -rf "${prefix}"
  mkdir -p "${prefix}"

  # 解出到 prefix（会包含 /usr/local/mysql 结构）
  if ! (cd "${prefix}" && cat "${payload}" | gunzip -dc | cpio -idm >/dev/null); then
    echo "[MySQL] 解包 Payload 失败：${payload}" >&2
    exit 1
  fi

  # 提取出 mysql 目录到 prefix_root/mysql
  # 通常路径：${prefix}/usr/local/mysql/*
  if [[ -d "${prefix}/usr/local/mysql" ]]; then
    mv "${prefix}/usr/local/mysql" "${prefix}/mysql"
    rm -rf "${prefix}/usr"
  fi

  # 最终统一：${BIN}/mysql/bin/...
  if [[ -d "${prefix}/mysql" ]]; then
    rm -rf "${prefix}/bin" "${prefix}/lib" "${prefix}/share"
    mv "${prefix}/mysql/bin" "${prefix}/bin"
    mv "${prefix}/mysql/lib" "${prefix}/lib"
    mv "${prefix}/mysql/share" "${prefix}/share" || true
    rm -rf "${prefix}/mysql"
  fi

  unquarantine "${prefix}"
}

# ---------- golang-migrate CLI (no brew) ----------
install_migrate() {
  local out="${BIN}/${MIGRATE_CLI}"
  if [[ -x "${out}" ]]; then
    log "migrate already installed: ${out}"
    return
  fi

  mkdir -p "${DIST}"

  local plat="darwin"
  local arch="amd64"
  local sha=""

  if [[ "${ARCH}" == "arm64" ]]; then
    arch="arm64"
    sha="${MIGRATE_SHA256_DARWIN_ARM64:-}"
  else
    arch="amd64"
    sha="${MIGRATE_SHA256_DARWIN_AMD64:-}"
  fi

  # Asset naming on releases: migrate.<platform>-<arch>.tar.gz
  # Example: migrate.darwin-arm64.tar.gz
  local name="${MIGRATE_CLI}.${plat}-${arch}.tar.gz"
  local url="https://github.com/golang-migrate/migrate/releases/download/v${MIGRATE_VER}/${name}"
  local tgz="${DIST}/${name}"

  curl_dl "${url}" "${tgz}"
  sha256_check "${tgz}" "${sha}"

  rm -rf "${DIST}/migrate-extract"
  mkdir -p "${DIST}/migrate-extract"
  tar -xzf "${tgz}" -C "${DIST}/migrate-extract"

  # The tar contains the binary named 'migrate'
  cp -f "${DIST}/migrate-extract/${MIGRATE_CLI}" "${out}"
  chmod +x "${out}"
  unquarantine "${out}"

  log "migrate installed: ${out}"
}

# ---------- configs ----------
gen_configs() {
  mkdir -p "${ETC}"

  local redis_conf="${ETC}/redis.conf"
  if [[ ! -f "${redis_conf}" ]]; then
    cat > "${redis_conf}" <<EOF
bind 127.0.0.1
port 6379
protected-mode yes
daemonize no
appendonly yes
dir ${DATA}/redis
logfile ${DATA}/logs/redis.log
EOF
  fi

  local mycnf="${ETC}/my.cnf"
  if [[ ! -f "${mycnf}" ]]; then
    cat > "${mycnf}" <<EOF
[mysqld]
bind-address=127.0.0.1
port=3306
max_connections=300
sql_mode=STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION
slow_query_log=1
long_query_time=0.2
log_error=${DATA}/logs/mysql.err
datadir=${DATA}/mysql
EOF
  fi
}

# ---------- main ----------
log "Setup binaries into ${BIN} (no brew, no system install)"

install_redis
install_mysql
install_migrate
# 可选
install_minio
# install_jdk && install_kafka   # 需要 Kafka 再打开
# install_jaeger                  # 建议你先固定 URL 再打开

gen_configs

log "Done. Next: scripts/dev_start.sh"