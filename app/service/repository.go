package service

import (
	"../../app/libs"
	"errors"
	"fmt"
	"github.com/astaxie/beego"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type repositoryService struct{}

// 返回一个仓库对象
func (this *repositoryService) GetRepoByProjectId(projectId int) (*Repository, error) {
	project, err := ProjectService.GetProject(projectId)
	if err != nil {
		return nil, err
	}
	return OpenRepository(project.Domain)
}

// 获取某项目代码库的标签列表
func (this *repositoryService) GetTags(projectId int, limit int) ([]string, error) {
	repo, err := this.GetRepoByProjectId(projectId)
	if err != nil {
		return nil, err
	}
	repo.Pull()
	list, err := repo.GetTags()
	if err != nil {
		return nil, err
	}
	if len(list) > limit {
		list = list[0:limit]
	}
	return list, nil
}

func (this *repositoryService) GetBranchs(projectId int, limit int) ([]string, error) {
	repo, err := this.GetRepoByProjectId(projectId)
	if err != nil {
		return nil, err
	}
	repo.Pull()
	list, err := repo.GetBranchs()
	if err != nil {
		return nil, err
	}
	if len(list) > limit {
		list = list[0:limit]
	}
	return list, nil
}

// 克隆git仓库
func (this *repositoryService) CloneRepo(url string, dst string) error {
	beego.Trace("CloneRepo,: ", url, dst)
	out, stderr, err := libs.ExecCmd("git", "clone", url, dst)
	debug("out", out)
	debug("stderr", stderr)
	debug("err", err)
	if err != nil {
		return concatenateError(err, stderr)
	}
	return nil
}

type SortTag struct {
	data []string
}

func (t *SortTag) Len() int {
	return len(t.data)
}
func (t *SortTag) Swap(i, j int) {
	t.data[i], t.data[j] = t.data[j], t.data[i]
}
func (t *SortTag) Less(i, j int) bool {
	return libs.VerCompare(t.data[i], t.data[j]) == 1
}
func (t *SortTag) Sort() []string {
	sort.Sort(t)
	return t.data
}

type Repository struct {
	Path string
}

func OpenRepository(repoPath string) (*Repository, error) {
	repoPath = GetProjectPath(repoPath)
	repoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	} else if !libs.IsDir(repoPath) {
		return nil, errors.New("no such file or directory")
	}

	return &Repository{Path: repoPath}, nil
}

// 拉取代码
func (repo *Repository) Pull() error {
	_, stderr, err := libs.ExecCmdDir(repo.Path, "git", "pull")
	if err != nil {
		return concatenateError(err, stderr)
	}
	return nil
}

// 获取tag列表
func (repo *Repository) GetTags() ([]string, error) {
	stdout, stderr, err := libs.ExecCmdDir(repo.Path, "git", "tag", "-l")
	if err != nil {
		return nil, concatenateError(err, stderr)
	}
	tags := strings.Split(stdout, "\n")
	tags = tags[:len(tags)-1]
	/*if len(tags) > 20 {
		tags = tags[len(tags)-20:]
	}

	var revTags []string
	for a := len(tags) - 1; a > 0; a-- {
		revTags = append(revTags, tags[a])
	}

	return revTags, nil*/

	so := &SortTag{data: tags}
	return so.Sort(), nil
}

func (repo *Repository) GetBranchs() ([]string, error) {
	stdout, stderr, err := libs.ExecCmdDir(repo.Path, "git", "branch", "-r")
	if err != nil {
		return nil, concatenateError(err, stderr)
	}
	tags := strings.Split(stdout, "\n")
	tags = tags[:len(tags)-1]
	if len(tags) > 20 {
		tags = tags[len(tags)-20:]
	}

	var revTags []string
	for a := len(tags) - 1; a > 0; a-- {
		revTags = append(revTags, tags[a])
	}

	return revTags, nil

	/*so := &SortTag{data: tags}
	return so.Sort(), nil*/
}

func (this *repositoryService) CreateTag(projectId int, projectName string, branchName string) error {
	repo, err := this.GetRepoByProjectId(projectId)
	list, _ := repo.GetTags()
	var latest = ""
	if len(list) > 0 {
		latest = list[0]
	}
	beego.Trace("Latest tag:", latest)
	if err != nil {
		return err
	}
	now := time.Now().Format("20060102")
	tagname := projectName + "_" + now
	if strings.Contains(latest, now) {
		index := strings.Index(latest, now) + len(now)
		nowVer := latest[index : index+2]
		nowVerNum, _ := strconv.Atoi(nowVer)
		nowVerNum += 1
		tagname += fmt.Sprintf("%02d", nowVerNum)
	} else {
		tagname += "00"
	}

	if strings.Contains(latest, "_v2") {
		substr := latest[strings.Index(latest, "_v2")+1:]
		versions := strings.Split(substr, ".")
		incre, _ := strconv.Atoi(versions[2])
		incre += 1
		tagname += "_v2." + versions[1] + "." + strconv.Itoa(incre)
	} else {
		tagname += "_v2.0.0"
	}
	beego.Trace("tagname:", tagname)

	cmd := "git tag " + tagname + " " + branchName + " && git push origin " + tagname
	_, stderr, err := libs.ExecCmdDir(repo.Path, "/bin/bash", "-c", cmd)
	if err != nil {
		return concatenateError(err, stderr)
	}

	return nil
}

// 获取两个版本之间的修改日志
func (repo *Repository) GetChangeLogs(startVer, endVer string) ([]string, error) {
	// git log --pretty=format:"%cd %cn: %s" --date=iso v1.8.0...v1.9.0
	stdout, stderr, err := libs.ExecCmdDir(repo.Path, "git", "log", "--pretty=format:%cd %cn: %s", "--date=iso", startVer+"..."+endVer)
	if err != nil {
		return nil, concatenateError(err, stderr)
	}

	logs := strings.Split(stdout, "\n")
	return logs, nil
}

// 获取两个版本之间的差异文件列表
func (repo *Repository) GetChangeFiles(startVer, endVer string, onlyFile bool) ([]string, error) {
	// git diff --name-status -b v1.8.0 v1.9.0
	param := "--name-status"
	if onlyFile {
		param = "--name-only"
	}
	stdout, stderr, err := libs.ExecCmdDir(repo.Path, "git", "diff", param, "-b", startVer, endVer)
	if err != nil {
		return nil, concatenateError(err, stderr)
	}
	lines := strings.Split(stdout, "\n")
	return lines[:len(lines)-1], nil
}

// 获取两个版本间的新增或修改的文件数量
func (repo *Repository) GetDiffFileCount(startVer, endVer string) (int, error) {
	cmd := "git diff --name-status -b " + startVer + " " + endVer + " |grep -v ^D |wc -l"
	stdout, stderr, err := libs.ExecCmdDir(repo.Path, "/bin/bash", "-c", cmd)
	if err != nil {
		return 0, concatenateError(err, stderr)
	}
	count, _ := strconv.Atoi(strings.TrimSpace(stdout))
	return count, nil
}

// 导出版本到tar包
func (repo *Repository) Export(startVer, endVer string, filename string) error {
	// git archive --format=tar.gz $endVer $(git diff --name-status -b $beginVer $endVer |grep -v ^D |grep -v Upgrade/ |awk '{print $2}') -o $tmpFile

	cmd := ""
	if startVer == "" {
		cmd = "git archive --format=tar " + endVer + " | gzip > " + filename
	} else {
		cmd = "git archive --format=tar " + endVer + " $(git diff --name-status -b " + startVer + " " + endVer + "|grep -v ^D |awk '{print $2}') | gzip > " + filename
	}
	beego.Trace("cmd:", cmd)

	_, stderr, err := libs.ExecCmdDir(repo.Path, "/bin/bash", "-c", cmd)

	if err != nil {
		return concatenateError(err, stderr)
	}
	return nil
}
