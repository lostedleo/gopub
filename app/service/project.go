package service

import (
	"../../app/entity"
	"os"
)

type projectService struct{}

// 表名
func (this *projectService) table() string {
	return tableName("project")
}

// 获取一个项目信息
func (this *projectService) GetProject(id int) (*entity.Project, error) {
	project := &entity.Project{}
	project.Id = id
	if err := o.Read(project); err != nil {
		return nil, err
	}
	return project, nil
}

// 获取所有项目
func (this *projectService) GetAllProject(filters ...interface{}) ([]entity.Project, int64) {
	return this.GetList(1, -1, filters...)
}

// 获取项目列表
func (this *projectService) GetList(page, pageSize int, filters ...interface{}) ([]entity.Project, int64) {
	var list []entity.Project
	offset := 0
	if pageSize == -1 {
		pageSize = 100000
	} else {
		offset = (page - 1) * pageSize
		if offset < 0 {
			offset = 0
		}
	}
	query := o.QueryTable(this.table())

	if len(filters) > 0 {

		length := len(filters)

		for k := 0; k < length; k += 2 {
			_, ok := filters[k].(string)
			if !ok {
				continue
			}
			v := filters[k+1]
			query = query.Filter(filters[k].(string), v)
		}
	}
	count, _ := query.Count()
	query.Offset(offset).Limit(pageSize).All(&list)
	return list, count
}

// 获取项目总数
func (this *projectService) GetTotal() (int64, error) {
	return o.QueryTable(this.table()).Count()
}

// 添加项目
func (this *projectService) AddProject(project *entity.Project) (int64, error) {
	pId, err := o.Insert(project)
	return pId, err
}

// 更新项目信息
func (this *projectService) UpdateProject(project *entity.Project, fields ...string) error {
	_, err := o.Update(project, fields...)
	return err
}

// 删除一个项目
func (this *projectService) DeleteProject(projectId int) error {
	project, err := this.GetProject(projectId)
	if err != nil {
		return err
	}
	// 删除目录
	path := GetProjectPath(project.Domain)
	os.RemoveAll(path)
	// 环境配置
	if envList, err := EnvService.GetEnvListByProjectId(project.Id); err != nil {
		for _, env := range envList {
			EnvService.DeleteEnv(env.Id)
		}
	}
	// 删除任务
	TaskService.DeleteByProjectId(project.Id)
	// 删除项目
	o.Delete(project)
	return nil
}

// 克隆某个项目的仓库
func (this *projectService) CloneRepo(projectId int) error {
	project, err := ProjectService.GetProject(projectId)
	if err != nil {
		return err
	}

	err = RepositoryService.CloneRepo(project.RepoUrl, GetProjectPath(project.Domain))
	if err != nil {
		project.Status = -1
		project.ErrorMsg = err.Error()
	} else {
		project.Status = 1
	}
	ProjectService.UpdateProject(project, "Status", "ErrorMsg")

	return err
}
