package database

import (
	"UnlockEdv2/src/models"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
	"unicode"

	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/gormlite"
	"github.com/teambition/rrule-go"

	"github.com/go-playground/validator/v10"
	"github.com/pressly/goose/v3"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DB struct{ *gorm.DB }

func NewDB(db *gorm.DB) *DB {
	return &DB{db}
}

func ValidateAlphaNumSpace(fl validator.FieldLevel) bool {
	for _, char := range fl.Field().String() {
		if !unicode.IsDigit(char) && !unicode.IsLetter(char) && !unicode.IsSpace(char) {
			return false
		}
	}
	return true
}

var Validate = sync.OnceValue(func() *validator.Validate {
	Ins := validator.New(validator.WithRequiredStructEnabled())
	err := Ins.RegisterValidation("alphanumspace", ValidateAlphaNumSpace, false)
	if err != nil {
		log.Fatalf("Failed to register custom validation: %v", err)
	}
	return Ins
})

func InitDB(isTesting bool) *DB {
	var gormDb *gorm.DB
	var err error
	if isTesting {
		gormDb, err = gorm.Open(gormlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			log.Fatal("Failed to connect to SQLite database:", err)
		}
		log.Println("Connected to the SQLite database in memory")
		MigrateTesting(gormDb)
	} else {
		dsn := os.Getenv("APP_DSN")
		if dsn == "" {
			dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=allow",
				os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
		}
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			log.Fatalf("Failed to open database connection: %v", err)
		}
		if err := db.Ping(); err != nil {
			log.Fatalf("Failed to ping database: %v", err)
		}
		log.Println("Running up migrations...")
		migrationDir := os.Getenv("MIGRATION_DIR")
		if migrationDir == "" {
			migrationDir = "migrations"
		}
		if err := goose.Up(db, migrationDir); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		gormDb, err = gorm.Open(postgres.New(postgres.Config{
			Conn: db,
		}), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to PostgreSQL database using GORM: %v", err)
		}
		log.Println("Connected to the PostgreSQL database via GORM")
	}
	DB := &DB{gormDb}
	DB.SeedDefaultData(isTesting)
	if isTesting {
		DB.SeedTestData()
	}
	return DB
}

func MigrateTesting(db *gorm.DB) {
	var TableList = []interface{}{
		&models.Role{},
		&models.User{},
		&models.ProviderPlatform{},
		&models.ProviderUserMapping{},
		&models.HelpfulLink{},
		&models.Course{},
		&models.Program{},
		&models.ProgramTag{},
		&models.ProgramSection{},
		&models.ProgramSectionEvent{},
		&models.ProgramSectionEnrollment{},
		&models.ProgramSectionEventOverride{},
		&models.ProgramSectionEventAttendance{},
		&models.Milestone{},
		&models.Outcome{},
		&models.Activity{},
		&models.OidcClient{},
		&models.ProgramFavorite{},
		&models.Facility{},
		&models.OpenContentProvider{},
		&models.OpenContentUrl{},
		&models.OpenContentActivity{},
		&models.CronJob{},
		&models.RunnableTask{},
		&models.Library{},
		&models.Video{},
		&models.VideoDownloadAttempt{},
		&models.OpenContentFavorite{},
		&models.UserEnrollment{},
		&models.UserCourseActivityTotals{},
	}
	for _, table := range TableList {
		log.Printf("Migrating %T table...", table)
		if err := db.AutoMigrate(table); err != nil {
			log.Fatal("Failed to migrate table: ", err)
		}
	}
}

func (db *DB) SeedDefaultData(isTesting bool) {
	var count int64
	if err := db.Model(models.User{}).Where("id = ?", fmt.Sprintf("%d", 1)).Count(&count).Error; err != nil {
		log.Fatal("db transaction failed getting admin user")
	}
	if count == 0 {
		if err := db.Model(models.Facility{}).Where("id = ?", fmt.Sprintf("%d", 1)).Count(&count).Error; err != nil {
			log.Fatal("db transaction failed getting default facility")
		}
		if isTesting {
			roles := []models.Role{{Name: "admin"}, {Name: "student"}, {Name: "system_admin"}}
			for _, role := range roles {
				if err := db.Create(&role).Error; err != nil {
					log.Fatalf("Failed to create role: %v", err)
				}
			}
		}
		defaultFacility := models.Facility{
			Name:     "Default",
			Timezone: "America/Chicago",
		}
		log.Printf("Creating facility: %v", defaultFacility)
		if err := db.Create(&defaultFacility).Error; err != nil {
			log.Fatalf("Failed to create user: %v", err)
		}
		user := models.User{
			Username:   "SuperAdmin",
			NameFirst:  "Super",
			NameLast:   "Admin",
			Email:      "admin@unlocked.v2",
			Role:       models.SystemAdmin,
			FacilityID: 1,
		}
		log.Printf("Creating user: %v", user)
		log.Println("Make sure to sync the Kratos instance if you are freshly migrating")
		if err := db.Create(&user).Error; err != nil {
			log.Fatalf("Failed to create user: %v", err)
		}

		links := []models.HelpfulLink{}
		if err := json.Unmarshal([]byte(defaultLeftMenuLinks), &links); err != nil {
			log.Fatalf("Failed to unmarshal default left menu links: %v", err)
		}
		if err := db.Create(&links).Error; err != nil {
			log.Fatalf("Failed to create left menu links: %v", err)
		}
		openContent := []models.OpenContentProvider{{Title: models.Kiwix, Url: models.KiwixLibraryUrl, CurrentlyEnabled: true, ThumbnailUrl: models.KiwixThumbnailURL, Description: models.Kiwix},
			{Title: models.Youtube, Url: models.YoutubeApi, CurrentlyEnabled: true, ThumbnailUrl: models.YoutubeThumbnail, Description: models.YoutubeDescription}}
		for idx := range openContent {
			if err := db.Create(&openContent[idx]).Error; err != nil {
				log.Fatalf("Failed to create kiwix open content provider: %v", err)
			}
		}
	}
}

const defaultLeftMenuLinks = `[{
			"title": "Unlocked Labs",
			"description": "Unlocked Labs website",
			"url": "https://unlockedlabs.org",
			"visibility_status": true,
	        "thumbnail_url": "https://unlockedlabs.org/favicon.ico",
			"open_content_provider_id": 1,
			"facility_id": 1
		},
	    {
	    "title": "Google",
	    "description": "Google search engine",
	    "url": "https://www.google.com",
	    "visibility_status": true,
	    "open_content_provider_id": 1,
	    "thumbnail_url": "https://www.google.com/favicon.ico",
	    "facility_id": 1
	    }]`

func (db *DB) SeedTestData() {
	facilitiesFile, err := os.ReadFile("test_data/facilities.json")
	if err != nil {
		log.Fatalf("Failed to read test data: %v", err)
	}
	var facilities []models.Facility
	if err := json.Unmarshal(facilitiesFile, &facilities); err != nil {
		log.Fatalf("Failed to unmarshal test data: %v", err)
	}
	for i := range facilities {
		if err := db.Create(&facilities[i]).Error; err != nil {
			log.Fatalf("Failed to create facility: %v", err)
		}
	}
	platforms, err := os.ReadFile("test_data/provider_platforms.json")
	if err != nil {
		log.Fatalf("Failed to read test data: %v", err)
	}
	var platform []models.ProviderPlatform
	if err := json.Unmarshal(platforms, &platform); err != nil {
		log.Fatalf("Failed to unmarshal test data: %v", err)
	}
	for i := range platform {
		if i%2 == 0 {
			platform[i].OidcID = 1
		}
		if err := db.Create(&platform[i]).Error; err != nil {
			log.Fatalf("Failed to create platform: %v", err)
		}
	}
	oidcFile, err := os.ReadFile("test_data/oidc_client.json")
	if err != nil {
		log.Fatalf("Failed to read test data: %v", err)
	}
	var oidcClients []models.OidcClient
	if err := json.Unmarshal(oidcFile, &oidcClients); err != nil {
		log.Fatalf("Failed to unmarshal test data: %v", err)
	}
	for i := range oidcClients {
		oidcClients[i].ProviderPlatformID = 3
		if err := db.Create(&oidcClients[i]).Error; err != nil {
			log.Fatalf("Failed to create oidc: %v", err)
		}
	}
	users, err := os.ReadFile("test_data/users.json")
	if err != nil {
		log.Fatalf("Failed to read test data: %v", err)
	}
	var user []models.User
	if err := json.Unmarshal(users, &user); err != nil {
		log.Fatalf("Failed to unmarshal test data: %v", err)
	}
	for idx := range user {
		log.Printf("Creating user %s", user[idx].Username)
		if err := db.Create(&user[idx]).Error; err != nil {
			log.Fatalf("Failed to create user: %v", err)
		}
		//skip last user to be added for any mapping for testing a scenario
		if idx == len(user)-2 {
			continue
		}
		for i := range platform {
			mapping := models.ProviderUserMapping{
				UserID:             user[idx].ID,
				ProviderPlatformID: platform[i].ID,
				ExternalUsername:   user[idx].Username,
				ExternalUserID:     strconv.Itoa(idx),
			}
			if err = db.CreateProviderUserMapping(&mapping); err != nil {
				return
			}
		}
	}
	var courses []models.Course
	progs, err := os.ReadFile("test_data/courses.json")
	if err != nil {
		log.Fatalf("Failed to read test data: %v", err)
	}
	if err := json.Unmarshal(progs, &courses); err != nil {
		log.Fatalf("Failed to unmarshal test data: %v", err)
	}
	for idx := range courses {
		if err := db.Create(&courses[idx]).Error; err != nil {
			log.Fatalf("Failed to create course: %v", err)
		}
	}
	sections, err := createFacilityPrograms(db)
	if err != nil {
		log.Fatalf("Failed to create facility programs: %v", err)
	}

	var programs []models.Program
	if err := db.Find(&programs).Error; err != nil {
		log.Fatalf("Failed to create facility programs: %v", err)
	}

	programTags := []string{"eligible_good_time", "self-paced"}
	for idx := range programs {
		tag := models.ProgramTag{
			ProgramID: programs[idx].ID,
			Value:     programTags[rand.Intn(len(programTags))],
		}
		if err := db.Create(&tag).Error; err != nil {
			log.Fatalf("Failed to create program tag: %v", err)
		}
	}
	outcomes := []string{"completion", "grade", "certificate", "pathway_completion"}
	//added this here from seeder
	milestoneTypes := []models.MilestoneType{models.DiscussionPost, models.AssignmentSubmission, models.QuizSubmission, models.GradeReceived}
	var dbUsers []models.User
	if db.Find(&dbUsers).Error != nil {
		log.Fatalf("Failed to get users from db")
		return
	}
	openContent := []models.OpenContentProvider{{Title: models.Kiwix, Url: models.KiwixLibraryUrl, CurrentlyEnabled: true, ThumbnailUrl: models.KiwixThumbnailURL, Description: models.Kiwix},
		{Title: models.Youtube, Url: models.YoutubeApi, CurrentlyEnabled: true, ThumbnailUrl: models.YoutubeThumbnail, Description: models.YoutubeDescription}}
	for idx := range openContent {
		if err := db.Create(&openContent[idx]).Error; err != nil {
			log.Fatalf("Failed to create kiwix open content provider: %v", err)
		}
	}
	libraries, err := os.ReadFile("test_data/libraries.json")
	if err != nil {
		log.Fatalf("Failed to read test data: %v", err)
	}
	var library []models.Library
	if err := json.Unmarshal(libraries, &library); err != nil {
		log.Fatalf("Failed to unmarshal test data: %v", err)
	}
	videosJson, err := os.ReadFile("test_data/videos.json")
	if err != nil {
		log.Fatalf("Failed to read test data: %v", err)
	}
	var videos []models.Video
	if err := json.Unmarshal(videosJson, &videos); err != nil {
		log.Fatalf("Failed to unmarshal test data: %v", err)
	}
	var kwixID uint //get id for kwix
	if db.Model(&models.OpenContentProvider{}).Select("id").Where("title = ?", models.Kiwix).First(&kwixID).RowsAffected == 0 {
		log.Fatalf("Failed to get %s open_content_provider: %v", models.Kiwix, err)
	}
	var youtubeID uint
	if db.Model(&models.OpenContentProvider{}).Select("id").Where("title = ?", models.Youtube).First(&youtubeID).RowsAffected == 0 {
		log.Fatalf("Failed to get %s open_content_provider: %v", models.Kiwix, err)
	}
	var url models.OpenContentUrl
	var activity models.OpenContentActivity
	for i := range videos {
		user := dbUsers[uint(rand.Intn(len(dbUsers)))]
		videos[i].OpenContentProviderID = youtubeID
		if err := db.Create(&videos[i]).Error; err != nil {
			log.Fatalf("Failed to create video: %v", err)
		}
		videoViewerUrl := fmt.Sprintf("/viewer/videos/%d", videos[i].ID)
		url = models.OpenContentUrl{
			ContentURL: videoViewerUrl,
		}
		if err := db.Create(&url).Error; err != nil {
			log.Fatalf("Failed to create content url: %v", err)
		}
		for j, k := 0, rand.Intn(50); j < k; j++ {
			activity = models.OpenContentActivity{
				OpenContentProviderID: youtubeID,
				FacilityID:            user.FacilityID,
				UserID:                user.ID,
				ContentID:             videos[i].ID,
				OpenContentUrlID:      url.ID,
				RequestTS:             time.Now(),
			}
			if err := db.Create(&activity).Error; err != nil {
				log.Fatalf("Failed to create open content activity: %v", err)
			}
			time.Sleep(time.Millisecond * 1)
		}
		if i%3 == 0 { //just going to favorite every third video
			favoriteVideo := models.OpenContentFavorite{
				UserID:                user.ID,
				ContentID:             videos[i].ID,
				OpenContentProviderID: youtubeID,
			}
			if err := db.Create(&favoriteVideo).Error; err != nil {
				log.Fatalf("Failed to create favorite video: %v", err)
			}
		}
	}
	openContentUrlPrefixes := []string{"alpha-bravo", "sunny-breeze", "stormy-night", "crimson-sky", "electric-wave", "golden-hour", "starry-dream", "lunar-echo", "cosmic-dust", "silent-whisper", "ocean-tide", "shadow-flame", "emerald-haze", "velvet-sun", "fire-bolt", "thunder-cloud", "frozen-peak", "radiant-gem", "mystic-vortex", "crystal-shard", "obsidian-moon", "solar-wind", "arctic-light", "nebula-glow", "desert-spark", "forest-blaze", "phantom-frost", "twilight-glimmer", "vivid-flare", "prism-halo", "aurora-wave", "blazing-star", "icy-horizon", "jagged-dream", "vivid-shadow", "iron-bloom", "canyon-sky", "frost-spark"}
	for i := range library {
		user := dbUsers[uint(rand.Intn(len(dbUsers)))]
		library[i].OpenContentProviderID = kwixID
		if err := db.Create(&library[i]).Error; err != nil {
			log.Fatalf("Failed to create library: %v", err)
		}
		for j, k := 0, len(openContentUrlPrefixes); j < k; j++ {
			url = models.OpenContentUrl{
				ContentURL: fmt.Sprintf("/api/proxy/libraries/%d/content/%s", library[i].ID, openContentUrlPrefixes[j]),
			}
			if err := db.Create(&url).Error; err != nil {
				log.Fatalf("Failed to create library: %v", err)
			}
			for j, k := 0, rand.Intn(50); j < k; j++ {

				activity = models.OpenContentActivity{
					OpenContentProviderID: kwixID,
					FacilityID:            user.FacilityID,
					UserID:                user.ID,
					ContentID:             library[i].ID,
					OpenContentUrlID:      url.ID,
					RequestTS:             time.Now(),
				}
				if err := db.Create(&activity).Error; err != nil {
					log.Fatalf("Failed to create open content activity: %v", err)
				}
				if i%2 == 0 && j == 0 { //just the first one should be favorited
					libraryFavorite := models.OpenContentFavorite{
						UserID:                user.ID,
						ContentID:             library[i].ID,
						OpenContentProviderID: kwixID,
					}
					if err := db.Create(&libraryFavorite).Error; err != nil {
						log.Fatalf("Failed to create favorite library: %v", err)
					}
				}
				time.Sleep(time.Millisecond * 1)
			}
		}
	}

	events := []models.ProgramSectionEvent{}
	if err := db.Find(&events).Error; err != nil {
		log.Fatalf("Failed to get events from db")
	}
	for idx := range dbUsers {
		if idx == 2 {
			continue
		}
		for jdx := range courses {
			// all test courses are open_enrollment
			enrollment := models.UserEnrollment{
				CourseID: courses[jdx].ID,
				UserID:   dbUsers[idx].ID,
			}
			enrollment.CreatedAt = &courses[jdx].CreatedAt
			if err := db.Create(&enrollment).Error; err != nil {
				log.Printf("Failed to create enrollment milestone: %v", err)
				continue
			}
			startTime := 0
			for i := 0; i < 365; i++ {
				if rand.Intn(100)%2 == 0 {
					continue
				}
				randTime := rand.Intn(1000)
				// we want activity for the last year
				yearAgo := time.Now().AddDate(-1, 0, 0)
				time := yearAgo.AddDate(0, 0, i)
				activity := models.Activity{
					UserID:     dbUsers[idx].ID,
					CourseID:   courses[jdx].ID,
					Type:       "interaction",
					TotalTime:  int64(startTime + randTime),
					TimeDelta:  int64(randTime),
					ExternalID: strconv.Itoa(rand.Intn(1000)),
					CreatedAt:  time,
				}
				startTime += randTime
				if err := db.Create(&activity).Error; err != nil {
					log.Fatalf("Failed to create activity: %v", err)
				}
			}
			if rand.Float32() < 0.4 { // 40% chance to create an outcome

				outcome := models.Outcome{
					CourseID: courses[jdx].ID,
					UserID:   dbUsers[idx].ID,
					Type:     models.OutcomeType(outcomes[rand.Intn(len(outcomes))]),
				}
				if err := db.Create(&outcome).Error; err != nil {
					log.Fatalf("Failed to create outcome: %v", err)
				}
			} else {
				newMilestone := models.Milestone{
					CourseID:    courses[jdx].ID,
					IsCompleted: false,
					Type:        milestoneTypes[rand.Intn(len(milestoneTypes))],
					UserID:      dbUsers[idx].ID,
					ExternalID:  strconv.Itoa(rand.Intn(10000)),
				}
				if err := db.Create(&newMilestone).Error; err != nil {
					log.Printf("Failed to create milestone: %v", err)
				}
				log.Printf("Creating milestone for user %s", dbUsers[idx].Username)

			}
		}
		for kdx := range sections {
			if sections[kdx].FacilityID == dbUsers[idx].FacilityID {
				enrollment := models.ProgramSectionEnrollment{
					UserID:    dbUsers[idx].ID,
					SectionID: sections[kdx].ID,
				}
				if err := db.Create(&enrollment).Error; err != nil {
					log.Printf("Failed to create enrollment: %v", err)
				}
				log.Printf("Creating program enrollment for user %s", dbUsers[idx].Username)
			}
		}
		for kdx := range events {
			attendance := models.ProgramSectionEventAttendance{
				EventID: events[kdx].ID,
				UserID:  dbUsers[idx].ID,
				Date:    time.Now().Format("2006-01-02"),
			}
			if err := db.Create(&attendance).Error; err != nil {
				log.Printf("Failed to create attendance for user: %v", err)
			}
		}

	}
}

func createFacilityPrograms(db *DB) ([]models.ProgramSection, error) {
	facilities := []models.Facility{}
	if err := db.Find(&facilities).Error; err != nil {
		return nil, err
	}
	toReturn := make([]models.ProgramSection, 0)
	for idx := range facilities {
		prog := models.Program{
			Name:        "Program for facility: " + facilities[idx].Name,
			Description: "Testing program",
		}
		if err := db.Create(&prog).Error; err != nil {
			log.Fatalf("Failed to create program: %v", err)
		}
		for i := 0; i < 5; i++ {
			section := models.ProgramSection{
				FacilityID: facilities[idx].ID,
				ProgramID:  prog.ID,
			}
			if err := db.Create(&section).Error; err != nil {
				log.Fatalf("Failed to create program section: %v", err)
			}
			log.Println("Creating program section ", section.ID)
			toReturn = append(toReturn, section)
			daysMap := make(map[int]rrule.Weekday)
			daysMap[0] = rrule.TU
			daysMap[1] = rrule.WE
			daysMap[2] = rrule.TH
			daysMap[3] = rrule.FR
			daysMap[4] = rrule.SA
			daysMap[5] = rrule.SU
			daysMap[6] = rrule.MO
			rule, err := rrule.NewRRule(rrule.ROption{
				Freq:      rrule.WEEKLY,
				Dtstart:   time.Now().Add(time.Duration(time.Month(i))),
				Count:     100,
				Byweekday: []rrule.Weekday{daysMap[rand.Intn(7)]},
			})
			if err != nil {
				log.Fatalf("Failed to create rrule: %v", err)
			}
			event := models.ProgramSectionEvent{
				SectionID:      section.ID,
				RecurrenceRule: rule.String(),
				Location:       "TBD",
				Duration:       "1h0m0s",
			}
			if err := db.Create(&event).Error; err != nil {
				log.Fatalf("Failed to create event: %v", err)
			}
		}
	}
	return toReturn, nil
}
