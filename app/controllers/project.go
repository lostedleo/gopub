package controllers

import (
	"fmt"
	"github.com/astaxie/beego"
	"../../app/entity"
	"../../app/libs"
	"../../app/service"
	"strconv"
	"strings"
	"bytes"
)

type ProjectController struct {
	BaseController
}

// 项目列表
func (this *ProjectController) List() {
	page, _ := strconv.Atoi(this.GetString("page"))
	if page < 1 {
		page = 1
	}
	beego.ReadFromRequest(&this.Controller)

	//var roleList []entity.Role
	roleList := this.auth.GetUser().RoleList

	pIds := make([]int, 0)
	for _, role := range roleList {
		if role.ProjectIds != "" {
			pIdArr := strings.Split(role.ProjectIds, ",")
			for _, v := range pIdArr{
				pId, _ := strconv.Atoi(v)
				pIds = append(pIds, pId)
			}
		}
	}

	filter := make([]interface{}, 0, 2)
	if len(pIds) > 0 {
		filter = append(filter, "id__in", pIds)
	} else {
		filter = append(filter, "id", 0)
	}

	list, count := service.ProjectService.GetList(page, this.pageSize, filter...)

	this.Data["count"] = count
	this.Data["list"] = list
	this.Data["pageBar"] = libs.NewPager(page, int(count), this.pageSize, beego.URLFor("ProjectController.List"), true).ToString()
	this.Data["pageTitle"] = "项目列表"
	this.display()
}

// 添加项目
func (this *ProjectController) Add() {

	if this.isPost() {
		p := &entity.Project{}
		p.Name = this.GetString("project_name")
		p.Domain = this.GetString("project_domain")
		p.RepoUrl = this.GetString("repo_url")
		p.AgentId, _ = this.GetInt("agent_id")
		p.IgnoreList = this.GetString("ignore_list")
		p.BeforeShell = this.GetString("before_shell")
		p.AfterShell = this.GetString("after_shell")
		p.TaskReview, _ = this.GetInt("task_review")
		if v, _ := this.GetInt("create_verfile"); v > 0 {
			p.CreateVerfile = 1
		} else {
			p.CreateVerfile = 0
		}
		p.VerfilePath = strings.Replace(this.GetString("verfile_path"), ".", "", -1)

		p.SendMail, _ = this.GetInt("send_mail")
		p.MailTplId, _ = this.GetInt("mail_tpl_id")

		if err := this.validProject(p); err != nil {
			this.showMsg(err.Error(), MSG_ERR)
		}

		pId, err := service.ProjectService.AddProject(p)
		this.checkError(err)

		roleList := this.auth.GetUser().RoleList

		for _, role := range roleList {
			role, err := service.RoleService.GetRole(role.Id)
			this.checkError(err)
			pIdStr := strconv.FormatInt(pId, 10)
			pIds := bytes.Buffer{}
			if role.ProjectIds != "" {
				pIds.WriteString(role.ProjectIds)
				pIds.WriteString(",")
			}
			pIds.WriteString(pIdStr)
			role.ProjectIds = pIds.String()
			err = service.RoleService.UpdateRole(role, "ProjectIds")
			this.checkError(err)
		}

		// 克隆仓库
		go service.ProjectService.CloneRepo(p.Id)

		service.ActionService.Add("add_project", this.auth.GetUserName(), "project", p.Id, "")

		this.redirect(beego.URLFor("ProjectController.List"))
	}

	agentList, err := service.ServerService.GetAgentList(1, -1)
	this.checkError(err)
	this.Data["pageTitle"] = "添加项目"
	this.Data["agentList"] = agentList
	this.Data["mailTplList"], _ = service.MailService.GetMailTplList()
	this.display()
}

// 编辑项目
func (this *ProjectController) Edit() {
	id, _ := this.GetInt("id")
	p, err := service.ProjectService.GetProject(id)
	this.checkError(err)

	if this.isPost() {
		p.Name = this.GetString("project_name")
		p.Domain = this.GetString("project_domain")
		p.AgentId, _ = this.GetInt("agent_id")
		p.IgnoreList = this.GetString("ignore_list")
		p.BeforeShell = this.GetString("before_shell")
		p.AfterShell = this.GetString("after_shell")
		p.TaskReview, _ = this.GetInt("task_review")
		if p.Status == -1 {
			p.RepoUrl = this.GetString("repo_url")
		}
		if v, _ := this.GetInt("create_verfile"); v > 0 {
			p.CreateVerfile = 1
		} else {
			p.CreateVerfile = 0
		}
		p.VerfilePath = strings.Replace(this.GetString("verfile_path"), ".", "", -1)

		if p.TaskReview == 1 {
			p.SendMail, _ = this.GetInt("send_mail")
			p.MailTplId, _ = this.GetInt("mail_tpl_id")
		} else {
			p.SendMail = 0
			p.MailTplId = 0
		}

		if err := this.validProject(p); err != nil {
			this.showMsg(err.Error(), MSG_ERR)
		}

		err := service.ProjectService.UpdateProject(p, "Name", "Domain", "AgentId", "IgnoreList",
			"BeforeShell", "AfterShell", "RepoUrl", "CreateVerfile", "VerfilePath", "TaskReview",
			"SendMail", "MailTplId")
		this.checkError(err)

		service.ActionService.Add("edit_project", this.auth.GetUserName(), "project", p.Id, "")

		this.redirect(beego.URLFor("ProjectController.List"))
	}

	agentList, err := service.ServerService.GetAgentList(1, -1)
	this.checkError(err)



	this.Data["project"] = p
	this.Data["agentList"] = agentList
	mailTplList, _ := service.MailService.GetMailTplList()
	this.Data["mailTplList"] = mailTplList
	this.Data["pageTitle"] = "编辑项目"
	this.display()
}

// 删除项目
func (this *ProjectController) Del() {
	id, _ := this.GetInt("id")

	roleList, err := service.RoleService.GetAllRoles()
	this.checkError(err)
	for _, role := range roleList {
		role, err := service.RoleService.GetRole(role.Id)
		this.checkError(err)

		if role.ProjectIds == "" {
			continue
		}
		old_pIds := strings.Split(role.ProjectIds, ",")

		buf := bytes.Buffer{}
		for i := 0;i< len(old_pIds); i++ {
			pId,_ := strconv.Atoi(old_pIds[i])
			if pId != id {
				buf.WriteString(old_pIds[i])
				buf.WriteString(",")
			}
		}
		new_pIds := buf.String()

		if new_pIds != "" {
			new_pIds = SubString(new_pIds, 0, len(new_pIds)-1)
		}
		role.ProjectIds = new_pIds
		err = service.RoleService.UpdateRole(role, "ProjectIds")
		this.checkError(err)
	}

	err = service.ProjectService.DeleteProject(id)
	this.checkError(err)

	service.ActionService.Add("del_project", this.auth.GetUserName(), "project", id, "")

	this.redirect(beego.URLFor("ProjectController.List"))
}
//截取字符串
func SubString(str string,begin,length int) (substr string) {
	// 将字符串的转换成[]rune
	rs := []rune(str)
	lth := len(rs)

	// 简单的越界判断
	if begin < 0 {
		begin = 0
	}
	if begin >= lth {
		begin = lth
	}
	end := begin + length
	if end > lth {
		end = lth
	}

	// 返回子串
	return string(rs[begin:end])
}

// 重新克隆
func (this *ProjectController) Clone() {
	id, _ := this.GetInt("id")
	project, err := service.ProjectService.GetProject(id)
	this.checkError(err)
	if project.Status != -1 {
		this.showMsg("只能对克隆失败的项目操作.", MSG_ERR)
	}

	project.Status = 0
	service.ProjectService.UpdateProject(project, "Status")
	go service.ProjectService.CloneRepo(id)

	this.showMsg("", MSG_OK)
}

// 获取项目克隆状态
func (this *ProjectController) GetStatus() {
	id, _ := this.GetInt("id")
	project, _ := service.ProjectService.GetProject(id)

	out := make(map[string]interface{})
	out["status"] = project.Status
	out["error"] = project.ErrorMsg

	this.jsonResult(out)
}

// 验证提交
func (this *ProjectController) validProject(p *entity.Project) error {
	errorMsg := ""
	if p.Name == "" {
		errorMsg = "请输入项目名称"
	} else if p.Domain == "" {
		errorMsg = "请输入项目标识"
	} else if p.RepoUrl == "" {
		errorMsg = "请输入仓库地址"
	} else if p.AgentId == 0 {
		errorMsg = "请选择跳板机"
	} else {
		//agent, err := service.ServerService.GetServer(p.AgentId)
		//if err != nil {
		//	return err
		//}
		//addr := fmt.Sprintf("%s:%d", agent.Ip, agent.SshPort)
		//serv := libs.NewServerConn(addr, agent.SshUser, agent.SshKey)
		//workPath := fmt.Sprintf("%s/%s", agent.WorkDir, p.Domain)
		//
		//if err := serv.TryConnect(); err != nil {
		//	errorMsg = "无法连接到跳板机: " + err.Error()
		//} else if _, err := serv.RunCmd("mkdir -p " + workPath); err != nil {
		//	errorMsg = "无法创建跳板机工作目录: " + err.Error()
		//}
		//serv.Close()
	}

	if errorMsg != "" {
		return fmt.Errorf(errorMsg)
	}
	return nil
}
