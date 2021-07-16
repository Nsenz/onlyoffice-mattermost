package main

import (
	"dto"
	"encoders"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"text/template"
	"utils"
)

func (p *Plugin) editor(writer http.ResponseWriter, request *http.Request) {
	err := request.ParseForm()
	if err != nil {
		p.API.LogError("[ONLYOFFICE]: Editor error ", err.Error())
		return
	}

	var fileId string = request.PostForm.Get("fileid")
	var docKey string = p.generateDocKey(fileId)

	fileInfo, _ := p.API.GetFileInfo(fileId)

	userId, _ := request.Cookie("MMUSERID")
	user, _ := p.API.GetUser(userId.Value)

	var serverURL string = *p.API.GetConfig().ServiceSettings.SiteURL + "/" + utils.MMPluginApi

	temp := template.New("onlyoffice")
	bundlePath, _ := p.API.GetBundlePath()
	temp, _ = temp.ParseFiles(filepath.Join(bundlePath, "public/editor.html"))

	p.encoder = encoders.EncoderAES{}
	fileId, _ = p.encoder.Encode(fileId, p.internalKey)

	var config dto.Config = dto.Config{
		Document: dto.Document{
			FileType: fileInfo.Extension,
			Key:      docKey,
			Title:    fileInfo.Name,
			Url:      serverURL + "/download?fileId=" + fileId,
		},
		DocumentType: utils.GetFileType(fileInfo.Extension),
		EditorConfig: dto.EditorConfig{
			User: dto.User{
				Id:   userId.Value,
				Name: user.Username,
			},
			CallbackUrl: serverURL + "/callback?fileId=" + fileId,
		},
	}

	jwtString, _ := utils.JwtSign(config, []byte(p.configuration.DESJwt))

	config.Token = jwtString

	data := map[string]interface{}{
		"apijs":  p.configuration.DESAddress + utils.DESApijs,
		"config": config,
	}

	temp.ExecuteTemplate(writer, "editor.html", data)
}

func (p *Plugin) callback(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()
	response := "{\"error\": 0}"

	body := dto.CallbackBody{}
	json.NewDecoder(request.Body).Decode(&body)

	handler, exists := p.getCallbackHandler(&body)

	p.encoder = encoders.EncoderAES{}
	fileId, _ := p.encoder.Decode(query.Get("fileId"), p.internalKey)
	body.FileId = fileId

	if !exists {
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(500)
		writer.Write([]byte(response))
	}

	handler(&body)

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(200)
	writer.Write([]byte(response))
}

func (p *Plugin) download(writer http.ResponseWriter, request *http.Request) {
	query := request.URL.Query()

	p.encoder = encoders.EncoderAES{}
	fileId, _ := p.encoder.Decode(query.Get("fileId"), p.internalKey)
	fileContent, _ := p.API.GetFile(fileId)

	writer.Write(fileContent)
}

func (p *Plugin) generateDocKey(fileId string) string {
	fileInfo, err := p.API.GetFileInfo(fileId)
	if err != nil {
		return ""
	}

	post, _ := p.API.GetPost(fileInfo.PostId)

	var postUpdatedAt string = strconv.FormatInt(post.EditAt, 10)

	p.encoder = encoders.EncoderRC4{}
	docKey, encodeErr := p.encoder.Encode(fileId+"_"+postUpdatedAt, []byte(utils.RC4Key))

	if encodeErr != nil {
		p.API.LogError("[ONLYOFFICE] Document key generation problem: ", encodeErr.Error())
		return ""
	}
	return docKey
}

func (p *Plugin) getCallbackHandler(callbackBody *dto.CallbackBody) (func(body *dto.CallbackBody), bool) {
	docServerStatus := map[int]func(body *dto.CallbackBody){
		1: p.handleIsBeingEdited,
		2: p.handleSave,
		3: p.handleSavingError,
		4: p.handleNoChanges,
		6: p.handleSave,
		7: p.handleForcesavingError,
	}

	handler, exists := docServerStatus[callbackBody.Status]

	return handler, exists
}
