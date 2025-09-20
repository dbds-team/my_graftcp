# GitHub Actions Release Workflow 配置说明

## 概述

该workflow在以`v`开头的tag被推送时自动触发，构建多架构版本并创建GitHub Release，同时将代码同步到私有仓库的`private_build`分支。

## 功能特性

- ✅ 多架构构建 (x86_64, aarch64)
- ✅ 自动创建GitHub Release
- ✅ 生成压缩包并上传到Release
- ✅ 同步代码到私有仓库指定分支
- ✅ 使用最新版Actions插件

## 必需的Secrets配置

在GitHub仓库设置中添加以下Secrets：

### 1. PRIVATE_REPO_TOKEN
- **用途**: 访问私有仓库的权限
- **类型**: Personal Access Token (Classic)
- **权限**: 需要 `repo` 完整权限
- **获取方式**:
  1. 访问 GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
  2. 生成新token，选择 `repo` 权限
  3. 复制token到仓库Secrets

### 2. PRIVATE_REPO_SSH_KEY
- **用途**: SSH方式推送到私有仓库
- **类型**: SSH私钥
- **获取方式**:
  ```bash
  # 生成SSH密钥对
  ssh-keygen -t ed25519 -C "github-actions@yourdomain.com" -f ~/.ssh/private_repo_key

  # 将公钥添加到私有仓库的Deploy Keys
  cat ~/.ssh/private_repo_key.pub

  # 将私钥内容添加到Secrets
  cat ~/.ssh/private_repo_key
  ```

## 使用方法

### 创建Release
```bash
# 创建并推送tag
git tag v1.0.0
git push origin v1.0.0

# 或者直接在GitHub界面创建Release
```

### 构建产物

Workflow会自动生成以下文件并上传到Release：
- `graftcp-v1.0.0-linux-amd64.tar.gz` - x86_64架构版本
- `graftcp-v1.0.0-linux-arm64.tar.gz` - ARM64架构版本

每个压缩包包含：
- `graftcp` - 主程序
- `graftcp-local` - 本地代理
- `mgraftcp` - 管理工具
- `README.md` / `README.zh-CN.md` - 说明文档
- `COPYING` - 许可证
- `example-*` - 配置示例

### 私有仓库同步

代码会自动推送到 `git@github.com:dbds-team/my_graftcp.git` 的 `private_build` 分支，确保主线仓库从origin更新，私有仓库在独立分支上。

## Workflow详细说明

### 构建阶段 (build job)
1. **环境准备**: Ubuntu最新版
2. **交叉编译**: 安装ARM64交叉编译工具链
3. **代码构建**: 使用Makefile构建不同架构版本
4. **打包发布**: 创建tar.gz压缩包
5. **上传构件**: 保存到GitHub Actions artifacts

### 发布阶段 (release job)
1. **下载构件**: 获取所有构建产物
2. **创建Release**: 使用最新的softprops/action-gh-release@v2
3. **上传文件**: 将所有架构的压缩包上传到Release
4. **生成说明**: 自动生成Release Notes

### 同步阶段 (sync-to-private job)
1. **配置SSH**: 使用webfactory/ssh-agent@v0.9.0
2. **添加远程**: 配置私有仓库remote
3. **推送代码**: 强制推送到private_build分支

## 故障排除

### 常见问题

1. **构建失败**: 检查Makefile和依赖
2. **权限错误**: 确认GITHUB_TOKEN权限
3. **SSH失败**: 验证SSH密钥配置
4. **同步失败**: 检查私有仓库访问权限

### 调试建议

- 查看Actions运行日志
- 验证Secrets配置
- 测试SSH连接到私有仓库
- 确认tag格式正确 (v开头)

## 安全注意事项

- SSH私钥仅用于GitHub Actions，不要在其他地方使用
- 定期轮换Personal Access Token
- 私有仓库Deploy Key权限最小化
- 监控Actions使用情况