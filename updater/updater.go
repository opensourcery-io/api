package updater

import (
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"github.com/opensourcery-io/api/services"
	"github.com/google/go-github/github"
	"github.com/opensourcery-io/api/models"
	"strings"
	"github.com/golang/glog"
)

const (
	UPDATE_ACTION          = "update"
	PRINT_ACTION           = "print"
	DEFAULT_INDEX_FILEPATH = "./index.json"
	DEFAULT_ACTION         = UPDATE_ACTION
)

type ProjectsIndex map[string][]string

type Updater struct {
	Index           string // location of index file
	Action          string // print or write to firebase
	GithubService   *services.GithubService
	FirebaseService *services.FirebaseService
	LogosService    *services.LogosService
}

func NewUpdater(
	index, action string,
	ghs *services.GithubService,
	lgs *services.LogosService,
	fbs *services.FirebaseService) *Updater {

	return &Updater{
		Index:           index,
		Action:          action,
		GithubService:   ghs,
		LogosService:    lgs,
		FirebaseService: fbs,
	}
}

func NewDefaultUpdater(firebaseCredsFile string) *Updater {
	fbs, err := services.NewFirebaseService(firebaseCredsFile)
	if err != nil {
		panic("Failed to create updater")
	}
	return NewUpdater(
		DEFAULT_INDEX_FILEPATH,
		DEFAULT_ACTION,
		services.NewGithubService(),
		services.NewLogosApiService(),
		fbs,
	)
}

func (u *Updater) loadProjectsIndex() (*ProjectsIndex, error) {
	file, err := ioutil.ReadFile(u.Index)
	if err != nil {
		return nil, err
	}

	index := ProjectsIndex{}
	if err := yaml.Unmarshal(file, &index); err != nil {
		return nil, err
	}

	return &index, nil
}

func (u *Updater) updateProject(project string, labels []string, chn chan<- []*github.Issue) {
	glog.Infof("Processing %v", project)
	tokens := strings.SplitN(project, "/", 2)
	if len(tokens) < 2 {
		glog.Errorf("Failed to parse %v. Err: %v", project)
	}

	owner, repo := tokens[0], tokens[1]
	//logo, err := u.LogosService.Search(repo)
	//if err != nil {
	//	glog.Warningf("Failed to get logo for %v, continuing. Err: %v", logo, err)
	//}

	allIssues := make([]*github.Issue, 0)
	for _, label := range labels {
		issues, err := u.GithubService.GetIssuesWithLabels(owner, repo, []string{label})
		if err != nil {
			glog.Errorf("Failed to get allIssues for %v and label %v. Err: %v", project, label, err)
			continue
		}

		allIssues = append(allIssues, issues...)
	}
	chn <- allIssues
	glog.Infof("Finished %v", project)
}

func (u *Updater) Update() ([]*models.Issue, error) {
	index, err := u.loadProjectsIndex()
	if err != nil {
		glog.Errorf("Failed to parse projects index. Err: %v", err)
	}
	glog.Infof("Loaded projects index with %v items", len(*index))

	allIssuesChn := make(chan []*github.Issue, len(*index))

	for project, labels := range *index {
		go u.updateProject(project, labels, allIssuesChn)
	}

	allIssues := make([]*github.Issue, 0)
	for i := 0; i< len(*index); i++ {
		issues := <- allIssuesChn
		allIssues = append(allIssues, issues...)
	}

	limit, err := u.GithubService.GetRateLimit()
	glog.Infof("Github Rate Limit: %v", limit)

	//fmt.Println(allIssuesChn)

	// TODO: Store to firebase
	issuesToStore := make([]*models.Issue, 0)

	return issuesToStore, nil
}
