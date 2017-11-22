package service

import (
	"../../app/entity"
	"../../app/libs"
	"html"
	"strings"
	"../../app/mail"
	"github.com/astaxie/beego"
)

type mailService struct{}

func (this *mailService) table() string {
	return tableName("mail_tpl")
}

func (this *mailService) AddMailTpl(tpl *entity.MailTpl) error {
	_, err := o.Insert(tpl)
	return err
}

func (this *mailService) DelMailTpl(id int) error {
	_, err := o.QueryTable(this.table()).Filter("id", id).Delete()
	return err
}

func (this *mailService) SaveMailTpl(tpl *entity.MailTpl) error {
	_, err := o.Update(tpl)
	return err
}

func (this *mailService) GetMailTpl(id int) (*entity.MailTpl, error) {
	tpl := &entity.MailTpl{}
	tpl.Id = id
	err := o.Read(tpl)
	return tpl, err
}

// 获取邮件模板列表
func (this *mailService) GetMailTplList() ([]entity.MailTpl, error) {
	var list []entity.MailTpl
	_, err := o.QueryTable(this.table()).OrderBy("-id").All(&list)
	return list, err
}

func (this *mailService) sendMailByTask(task *entity.Task,env *entity.Env,project *entity.Project) error {
	mailTpl, err := MailService.GetMailTpl(project.MailTplId)
	if err == nil {
		replace := make(map[string]string)
		replace["{project}"] = project.Name
		replace["{domain}"] = project.Domain
		if task.StartVer != "" {
			replace["{version}"] = task.StartVer + " - " + task.EndVer
		} else {
			replace["{version}"] = task.EndVer
		}

		replace["{env}"] = env.Name
		replace["{description}"] = libs.Nl2br(html.EscapeString(task.Message))
		replace["{changelogs}"] = libs.Nl2br(html.EscapeString(task.ChangeLogs))
		replace["{changefiles}"] = libs.Nl2br(html.EscapeString(task.ChangeFiles))

		subject := mailTpl.Subject
		content := mailTpl.Content

		for k, v := range replace {
			subject = strings.Replace(subject, k, v, -1)
			content = strings.Replace(content, k, v, -1)
		}

		mailTo := strings.Split(mailTpl.MailTo, "\n")
		mailCc := strings.Split(mailTpl.MailCc, "\n")
		if err := mail.SendMail(subject, content, mailTo, mailCc); err != nil {
			beego.Error("邮件发送失败：", err)
			//this.recordLog("task.publish", fmt.Sprintf("邮件发送失败：%v", err))
		}
		return err
	}
	return err
}