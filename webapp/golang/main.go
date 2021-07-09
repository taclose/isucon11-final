package main

import (
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pborman/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	SQLDirectory           = "../sql/"
	AssignmentsDirectory   = "../assignments/"
	AssignmentTmpDirectory = "../assignments/tmp/"
	SessionName            = "session"
)

type handlers struct {
	DB *sqlx.DB
}

func main() {
	e := echo.New()
	e.Debug = GetEnv("DEBUG", "") != ""
	e.Server.Addr = fmt.Sprintf(":%v", GetEnv("PORT", "7000"))
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("trapnomura"))))

	db, _ := GetDB(false)

	h := &handlers{
		DB: db,
	}

	// e.POST("/initialize", h.Initialize, h.IsLoggedIn, h.IsAdmin)
	e.POST("/initialize", h.Initialize)
	e.PUT("/phase", h.SetPhase, h.IsLoggedIn, h.IsAdmin)

	e.POST("/login", h.Login)
	API := e.Group("/api", h.IsLoggedIn)
	{
		usersAPI := API.Group("/users")
		{
			usersAPI.GET("/:userID/courses", h.GetRegisteredCourses)
			usersAPI.PUT("/:userID/courses", h.RegisterCourses)
			usersAPI.GET("/:userID/grades", h.GetGrades)
		}
		syllabusAPI := API.Group("/syllabus")
		{
			syllabusAPI.GET("", h.SearchCourses)
			syllabusAPI.GET("/:courseID", h.GetCourseDetail)
		}
		coursesAPI := API.Group("/courses")
		{
			coursesAPI.POST("", h.AddCourse, h.IsAdmin)
			coursesAPI.PUT("/:courseID/status", h.SetCourseStatus, h.IsAdmin)
			coursesAPI.GET("/:courseID/classes", h.GetClasses)
			coursesAPI.POST("/:courseID/classes", h.AddClass, h.IsAdmin)
			coursesAPI.POST("/:courseID/assignments/:assignmentID", h.SubmitAssignment)
			coursesAPI.GET("/:courseID/assignments/:assignmentID/export", h.DownloadSubmittedAssignment, h.IsAdmin)
			coursesAPI.POST("/:courseID/announcements", h.AddAnnouncements, h.IsAdmin)
		}
		announcementsAPI := API.Group("/announcements")
		{
			announcementsAPI.GET("", h.GetAnnouncementList)
			announcementsAPI.GET("/:announcementID", h.GetAnnouncementDetail)
		}
	}

	e.Logger.Error(e.StartServer(e.Server))
}

type InitializeResponse struct {
	Language string `json:"language"`
}

func (h *handlers) Initialize(c echo.Context) error {
	dbForInit, _ := GetDB(true)

	files := []string{
		"schema.sql",
		"test_data.sql",
	}
	for _, file := range files {
		data, err := ioutil.ReadFile(SQLDirectory + file)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("read sql file: %v", err))
		}
		if _, err := dbForInit.Exec(string(data)); err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("exec sql file: %v", err))
		}
	}

	res := InitializeResponse{
		Language: "go",
	}
	return c.JSON(http.StatusOK, res)
}

func (h *handlers) IsLoggedIn(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get(SessionName, c)
		if err != nil {
			return echo.ErrInternalServerError
		}
		if sess.IsNew {
			return echo.NewHTTPError(http.StatusUnauthorized, "You are not logged in.")
		}
		if _, ok := sess.Values["userID"]; !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "You are not logged in.")
		}

		return next(c)
	}
}

func (h *handlers) IsAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get(SessionName, c)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("get session: %v", err))
		}
		isAdmin, ok := sess.Values["isAdmin"]
		if !ok {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("get session value: %v", err))
		}
		if !isAdmin.(bool) {
			return echo.NewHTTPError(http.StatusForbidden, "You are not admin user.")
		}

		return next(c)
	}
}

type SetPhaseRequest struct {
	Phase    PhaseType `json:"phase"`
	Year     uint32    `json:"year"`
	Semester Semester  `json:"semester"`
}

func (h *handlers) SetPhase(c echo.Context) error {
	var req SetPhaseRequest
	if err := c.Bind(&req); err != nil {
		log.Println(err)
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	if req.Phase != PhaseRegistration && req.Phase != PhaseClass && req.Phase != PhaseResult {
		return echo.NewHTTPError(http.StatusBadRequest, "bad phase")
	}
	if req.Semester != FirstSemester && req.Semester != SecondSemester {
		return echo.NewHTTPError(http.StatusBadRequest, "bad semester")
	}

	if _, err := h.DB.Exec("TRUNCATE TABLE `phase`"); err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	if _, err := h.DB.Exec("INSERT INTO `phase` (`phase`, `year`, `semester`) VALUES (?, ?, ?)", req.Phase, req.Year, req.Semester); err != nil {
		log.Println(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusNoContent)
}

type GetGradesResponse struct {
	Summary      Summary        `json:"summary"`
	CourseGrades []*CourseGrade `json:"courses"`
}

type Summary struct {
	Credits int    `json:"credits"`
	GPT     uint32 `json:"gpt"`
}

type CourseGrade struct {
	ID     uuid.UUID `json:"id" db:"course_id"`
	Name   string    `json:"name" db:"name"`
	Credit uint8     `json:"credit" db:"credit"`
	Grade  string    `json:"grade" db:"grade"`
}

type RegisterCoursesRequestContent struct {
	ID string `json:"id"`
}

type RegisterCoursesRequest []RegisterCoursesRequestContent

type GetAnnouncementsResponse []GetAnnouncementResponse
type GetAnnouncementResponse struct {
	ID         uuid.UUID `json:"id"`
	CourseName string    `json:"course_name"`
	Title      string    `json:"title"`
	// MEMO: TODO: 既読機能
	// Unread     bool      `json:"unread"`
	CreatedAt int64 `json:"created_at"`
}

type GetAnnouncementDetailResponse struct {
	ID         uuid.UUID `json:"id"`
	CourseName string    `json:"course_name"`
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	CreatedAt  int64     `json:"created_at"`
}

type PostAnnouncementsRequest struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

type PostAnnouncementsResponse struct {
	ID uuid.UUID `json:"id"`
}

type TimeSlotResponse struct {
	Period    uint8  `json:"period"`
	DayOfWeek string `json:"day_of_week"`
}

type PhaseType string

const (
	PhaseRegistration PhaseType = "reg"
	PhaseClass        PhaseType = "class"
	PhaseResult       PhaseType = "result"
)

type Semester string

const (
	FirstSemester  Semester = "first"
	SecondSemester Semester = "second"
)

type Phase struct {
	Phase    PhaseType `json:"phase"`
	Year     uint32    `json:"year"`
	Semester Semester  `json:"semester"`
}

type UserType string

const (
	Student UserType = "student"
	Faculty UserType = "faculty"
)

type User struct {
	ID             uuid.UUID `db:"id"`
	Code           string    `db:"code"`
	Name           string    `db:"name"`
	HashedPassword []byte    `db:"hashed_password"`
	Type           UserType  `db:"type"`
}

type Course struct {
	ID          uuid.UUID     `db:"id"`
	Code        string        `db:"code"`
	Type        string        `db:"type"`
	Name        string        `db:"name"`
	Description string        `db:"description"`
	Credit      uint8         `db:"credit"`
	Classroom   string        `db:"classroom"`
	Capacity    sql.NullInt32 `db:"capacity"`
	TeacherID   uuid.UUID     `db:"teacher_id"`
	Keywords    string        `db:"keywords"`
}

type Schedule struct {
	ID        uuid.UUID `db:"id"`
	Period    uint8     `db:"period"`
	DayOfWeek string    `db:"day_of_week"`
	Semester  Semester  `db:"semester"`
	Year      uint32    `db:"year"`
}

type Class struct {
	ID             uuid.UUID `db:"id"`
	CourseID       uuid.UUID `db:"course_id"`
	Part           uint8     `db:"part"`
	Title          string    `db:"title"`
	Description    string    `db:"description"`
	AttendanceCode string    `db:"attendance_code"`
}

type Assignment struct {
	ID          uuid.UUID `db:"id"`
	ClassID     uuid.UUID `db:"class_id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	CreatedAt   time.Time `db:"created_at"`
}

type Announcement struct {
	ID         uuid.UUID `db:"id"`
	CourseName string    `db:"name"`
	Title      string    `db:"title"`
	Message    string    `db:"message"`
	CreatedAt  time.Time `db:"created_at"`
}

type SubmissionWithUserName struct {
	ID           uuid.UUID `db:"id"`
	UserID       uuid.UUID `db:"user_id"`
	UserName     string    `db:"user_name"`
	AssignmentID uuid.UUID `db:"assignment_id"`
	Name         string    `db:"name"`
	CreatedAt    time.Time `db:"created_at"`
}

type LoginRequest struct {
	Code     string `json:"code"`
	Password string `json:"password"`
}

func (h *handlers) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bind request: %v", err))
	}

	var user User
	err := h.DB.Get(&user, "SELECT * FROM `users` WHERE `code` = ?", req.Code)
	if err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusUnauthorized, "Code or Password is wrong.")
	} else if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if bcrypt.CompareHashAndPassword(user.HashedPassword, []byte(req.Password)) != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Code or Password is wrong.")
	}

	sess, err := session.Get(SessionName, c)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if s, ok := sess.Values["userID"].(string); ok {
		userID := uuid.Parse(s)
		if uuid.Equal(userID, user.ID) {
			return echo.NewHTTPError(http.StatusBadRequest, "You are already logged in.")
		}
	}

	sess.Values["userID"] = user.ID.String()
	sess.Values["userName"] = user.Name
	sess.Values["isAdmin"] = user.Type == Faculty
	sess.Options = &sessions.Options{
		Path:   "/",
		MaxAge: 3600,
	}

	if err := sess.Save(c.Request(), c.Response()); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

type GetRegisteredCourseResponse struct {
	ID          uuid.UUID          `json:"id"`
	Name        string             `json:"name"`
	TeacherName string             `json:"teacher"`
	Classroom   string             `json:"classroom"`
	Timeslots   []TimeSlotResponse `json:"timeslots"`
}

func (h *handlers) GetRegisteredCourses(context echo.Context) error {
	sess, err := session.Get(SessionName, context)
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if uuid.Equal(uuid.NIL, userID) {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	userIDParam := uuid.Parse(context.Param("userID"))
	if uuid.Equal(uuid.NIL, userIDParam) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid userID")
	}
	if !uuid.Equal(userID, userIDParam) {
		return echo.NewHTTPError(http.StatusForbidden, "invalid userID")
	}

	var phase Phase
	if err := h.DB.Get(&phase, "SELECT * FROM `phase`"); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	var courses []Course
	if err = h.DB.Select(&courses, "SELECT DISTINCT `courses`.* "+
		"FROM `courses` "+
		"JOIN `course_schedules` ON `courses`.`id` = `course_schedules`.`course_id` "+
		"JOIN `schedules` ON `course_schedules`.`schedule_id` = `schedules`.`id` "+
		"JOIN `registrations` ON `courses`.`id` = `registrations`.`course_id` "+
		"WHERE `schedules`.`year` = ? AND `schedules`.`semester` = ? AND `registrations`.`user_id` = ? AND `registrations`.`deleted_at` IS NULL", phase.Year, phase.Semester, userID); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	res := make([]GetRegisteredCourseResponse, 0, len(courses))
	for _, course := range courses {
		var teacher User
		if err := h.DB.Get(&teacher, "SELECT * FROM `users` WHERE `id` = ?", course.TeacherID); err != nil {
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}

		var schedules []Schedule
		if err := h.DB.Select(&schedules, "SELECT `schedules`.* "+
			"FROM `schedules` "+
			"JOIN `course_schedules` ON `schedules`.`id` = `course_schedules`.`schedule_id` "+
			"WHERE `course_schedules`.`course_id` = ?", course.ID); err != nil {
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}

		timeslotsRes := make([]TimeSlotResponse, 0, len(schedules))
		for _, schedule := range schedules {
			timeslotsRes = append(timeslotsRes, TimeSlotResponse{
				Period:    schedule.Period,
				DayOfWeek: schedule.DayOfWeek,
			})
		}

		res = append(res, GetRegisteredCourseResponse{
			ID:          course.ID,
			Name:        course.Name,
			TeacherName: teacher.Name,
			Classroom:   course.Classroom,
			Timeslots:   timeslotsRes,
		})
	}

	return context.JSON(http.StatusOK, res)
}

type RegisterCoursesErrorResponse struct {
	NotFoundCourse         []string    `json:"not_found_course,omitempty"`
	NotTakenRequiredCourse []uuid.UUID `json:"not_taken_required_course,omitempty"`
	CapacityExceeded       []uuid.UUID `json:"capacity_exceeded,omitempty"`
	TimeslotDuplicated     []uuid.UUID `json:"timeslot_duplicated,omitempty"`
}

func (h *handlers) RegisterCourses(context echo.Context) error {
	sess, err := session.Get(SessionName, context)
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if uuid.Equal(uuid.NIL, userID) {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	userIDParam := uuid.Parse(context.Param("userID"))
	if uuid.Equal(uuid.NIL, userIDParam) {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid userID")
	}
	if !uuid.Equal(userID, userIDParam) {
		return echo.NewHTTPError(http.StatusForbidden, "invalid userID")
	}

	var req RegisterCoursesRequest
	if err := context.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bind request: %v", err))
	}

	var phase Phase
	if err := h.DB.Get(&phase, "SELECT * FROM `phase`"); err != nil {
		context.Logger().Error(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	if phase.Phase != PhaseRegistration {
		return echo.NewHTTPError(http.StatusBadRequest, "not registration phase.")
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	// MEMO: SELECT ... FOR UPDATE は今のDB構造だとデッドロックする
	if _, err = tx.Exec("LOCK TABLES `registrations` WRITE, `courses` READ, `course_requirements` READ, `grades` READ, `schedules` READ, `course_schedules` READ"); err != nil {
		_ = tx.Rollback()
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	defer func() {
		_, _ = h.DB.Exec("UNLOCK TABLES")
	}()

	hasError := false
	var errors RegisterCoursesErrorResponse
	var courseList []Course
	for _, content := range req {
		courseID := uuid.Parse(content.ID)
		if courseID == nil {
			hasError = true
			errors.NotFoundCourse = append(errors.NotFoundCourse, content.ID)
			continue
		}

		var course Course
		if err := tx.Get(&course, "SELECT `courses`.* "+
			"FROM `courses` "+
			"JOIN `course_schedules` ON `courses`.`id` = `course_schedules`.`course_id` "+
			"JOIN `schedules` ON `schedules`.`id` = `course_schedules`.`schedule_id` "+
			"WHERE `courses`.`id` = ? AND `schedules`.`year` = ? AND `schedules`.`semester` = ?", courseID, phase.Year, phase.Semester); err == sql.ErrNoRows {
			hasError = true
			errors.NotFoundCourse = append(errors.NotFoundCourse, content.ID)
			continue
		} else if err != nil {
			_ = tx.Rollback()
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}

		// MEMO: すでに履修登録済みの科目は無視する
		var registerCount int
		if err := tx.Get(&registerCount, "SELECT COUNT(*) FROM `registrations` WHERE `course_id` = ? AND `user_id` = ?", course.ID, userID); err != nil {
			_ = tx.Rollback()
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}
		if registerCount > 0 {
			continue
		}

		courseList = append(courseList, course)
	}

	for _, course := range courseList {
		// MEMO: 前提講義バリデーション
		var requiredCourseIDList []uuid.UUID
		if err = tx.Select(&requiredCourseIDList, "SELECT `required_course_id` FROM `course_requirements` WHERE `course_id` = ?", course.ID); err != nil {
			_ = tx.Rollback()
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}
		for _, requiredCourseID := range requiredCourseIDList {
			var gradeCount int
			if err = tx.Get(&gradeCount, "SELECT COUNT(*) FROM `grades` WHERE `user_id` = ? AND `course_id` = ?", userID, requiredCourseID); err != nil {
				_ = tx.Rollback()
				log.Println(err)
				return context.NoContent(http.StatusInternalServerError)
			}
			if gradeCount == 0 {
				hasError = true
				errors.NotTakenRequiredCourse = append(errors.NotTakenRequiredCourse, course.ID)
				break
			}
		}

		// MEMO: 受講者数制限バリデーション
		var registerCount uint32
		if err = tx.Get(&registerCount, "SELECT COUNT(*) FROM `registrations` WHERE `course_id` = ?", course.ID); err != nil {
			_ = tx.Rollback()
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}
		if course.Capacity.Valid && registerCount >= uint32(course.Capacity.Int32) {
			hasError = true
			errors.CapacityExceeded = append(errors.CapacityExceeded, course.ID)
		}
	}

	if len(courseList) > 0 {
		// MEMO: スケジュールの重複バリデーション
		courseIDList := make([]uuid.UUID, 0, len(courseList))
		for _, course := range courseList {
			courseIDList = append(courseIDList, course.ID)
		}

		query := "SELECT `schedules`.* " +
			"FROM `schedules`" +
			"JOIN `course_schedules` ON `schedules`.`id` = `course_schedules`.`schedule_id` " +
			"WHERE `course_schedules`.`course_id` IN (?)"
		query, args, err := sqlx.In(query, courseIDList)
		if err != nil {
			_ = tx.Rollback()
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}

		var schedules []Schedule
		if err := tx.Select(&schedules, query, args...); err != nil {
			_ = tx.Rollback()
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}

		var registeredSchedules []Schedule
		if err := tx.Select(&registeredSchedules, "SELECT `schedules`.* "+
			"FROM `schedules` "+
			"JOIN `course_schedules` ON `schedules`.`id` = `course_schedules`.`schedule_id` "+
			"JOIN `registrations` ON `course_schedules`.`course_id` = `registrations`.`course_id` "+
			"WHERE `schedules`.`year` = ? AND `schedules`.`semester` = ? AND `registrations`.`user_id` = ? AND `registrations`.`deleted_at` IS NULL", phase.Year, phase.Semester, userID); err != nil {
			_ = tx.Rollback()
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}

		registeredSchedules = append(registeredSchedules, schedules...)

		for _, schedule1 := range schedules {
			for _, schedule2 := range registeredSchedules {
				if !uuid.Equal(schedule1.ID, schedule2.ID) && schedule1.Period == schedule2.Period && schedule1.DayOfWeek == schedule2.DayOfWeek {
					hasError = true
					errors.TimeslotDuplicated = append(errors.TimeslotDuplicated, schedule1.ID)
					break
				}
			}
		}
	}

	if hasError {
		_ = tx.Rollback()
		return context.JSON(http.StatusBadRequest, errors)
	}

	for _, course := range courseList {
		_, err = tx.Exec("INSERT INTO `registrations` (`course_id`, `user_id`, `created_at`) VALUES (?, ?, NOW(6))", course.ID, userID)
		if err != nil {
			_ = tx.Rollback()
			log.Println(err)
			return context.NoContent(http.StatusInternalServerError)
		}
	}

	if err = tx.Commit(); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	return context.NoContent(http.StatusOK)
}

func (h *handlers) GetGrades(context echo.Context) error {
	sess, err := session.Get(SessionName, context)
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if userID == nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	userIDParam := uuid.Parse(context.Param("userID"))
	if userIDParam == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid userID")
	}
	if !uuid.Equal(userID, userIDParam) {
		return echo.NewHTTPError(http.StatusForbidden, "invalid userID")
	}

	// MEMO: GradeテーブルとCoursesテーブルから、対象userIDのcourse_id/name/credit/gradeを取得
	var CourseGrades []CourseGrade
	query := "SELECT `course_id`, `name`, `credit`, `grade`" +
		"FROM `grades`" +
		"JOIN `courses` ON `grades`.`course_id` = `courses`.`id`" +
		"WHERE `user_id` = ?"
	if err := h.DB.Select(&CourseGrades, query, userID); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	var res GetGradesResponse
	var grade uint32
	var gpt uint32 = 0

	var credits int = 0
	if len(CourseGrades) > 0 {
		for _, coursegrade := range CourseGrades {
			res.CourseGrades = append(res.CourseGrades, &CourseGrade{
				ID:     coursegrade.ID,
				Name:   coursegrade.Name,
				Credit: coursegrade.Credit,
				Grade:  coursegrade.Grade,
			})

			switch coursegrade.Grade {
			case "S":
				grade = 4
			case "A":
				grade = 3
			case "B":
				grade = 2
			case "C":
				grade = 1
			case "D":
				grade = 0
			}
			credits += int(coursegrade.Credit)
			gpt += grade * uint32(coursegrade.Credit)
		}
	}

	res.Summary = Summary{
		Credits: credits,
		GPT:     gpt,
	}

	return context.JSON(http.StatusOK, res)
}

func (h *handlers) SearchCourses(context echo.Context) error {
	panic("implement me")
}

type GetCourseDetailResponse struct {
	ID              uuid.UUID                `json:"id"`
	Code            string                   `json:"code"`
	Type            string                   `json:"type"`
	Name            string                   `json:"name"`
	Description     string                   `json:"description"`
	Credit          uint8                    `json:"credit"`
	Classroom       string                   `json:"classroom"`
	Capacity        uint32                   `json:"capacity,omitempty"`
	Teacher         string                   `json:"teacher"`
	Keywords        string                   `json:"keywords"`
	Semester        Semester                 `json:"semester"`
	Year            uint32                   `json:"year"`
	Timeslots       []TimeSlotResponse       `json:"timeslots"`
	RequiredCourses []RequiredCourseResponse `json:"required_courses"`
}

type RequiredCourseResponse struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

func (h *handlers) GetCourseDetail(context echo.Context) error {
	courseID := uuid.Parse(context.Param("courseID"))
	if courseID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid courseID")
	}

	var course Course
	if err := h.DB.Get(&course, "SELECT * FROM `courses` WHERE `id` = ?", courseID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "No such course")
	} else if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	var teacher User
	if err := h.DB.Get(&teacher, "SELECT * FROM `users` WHERE `id` = ?", course.TeacherID); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	var schedules []Schedule
	if err := h.DB.Select(&schedules, "SELECT `schedules`.* "+
		"FROM `schedules` "+
		"JOIN `course_schedules` ON `schedules`.`id` = `course_schedules`.`schedule_id` "+
		"WHERE `course_schedules`.`course_id` = ?", course.ID); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	if len(schedules) == 0 {
		return context.NoContent(http.StatusInternalServerError)
	}

	var requiredCourses []Course
	if err := h.DB.Select(&requiredCourses, "SELECT `courses`.* "+
		"FROM `course_requirements` "+
		"JOIN `courses` ON `course_requirements`.`required_course_id` = `courses`.`id` "+
		"WHERE `course_requirements`.`course_id` = ?", course.ID); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	timeslotsRes := make([]TimeSlotResponse, 0, len(schedules))
	for _, schedule := range schedules {
		timeslotsRes = append(timeslotsRes, TimeSlotResponse{
			Period:    schedule.Period,
			DayOfWeek: schedule.DayOfWeek,
		})
	}

	requiredCoursesRes := make([]RequiredCourseResponse, 0, len(requiredCourses))
	for _, course := range requiredCourses {
		requiredCoursesRes = append(requiredCoursesRes, RequiredCourseResponse{
			ID:   course.ID,
			Name: course.Name,
		})
	}

	res := GetCourseDetailResponse{
		ID:              course.ID,
		Code:            course.Code,
		Type:            course.Type,
		Name:            course.Name,
		Description:     course.Description,
		Credit:          course.Credit,
		Classroom:       course.Classroom,
		Teacher:         teacher.Name,
		Keywords:        course.Keywords,
		Semester:        schedules[0].Semester,
		Year:            schedules[0].Year,
		Timeslots:       timeslotsRes,
		RequiredCourses: requiredCoursesRes,
	}

	if course.Capacity.Valid {
		res.Capacity = uint32(course.Capacity.Int32)
	}

	return context.JSON(http.StatusOK, res)
}

type AddCourseRequest struct {
	Code        string `json:"code"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Credit      int    `json:"credit"`
	Period      int    `json:"period"`
	DayOfWeek   string `json:"day_of_week"`
	Keywords    string `json:"keywords"`
}

type AddCourseResponse struct {
	ID uuid.UUID `json:"id"`
}

func (h *handlers) AddCourse(c echo.Context) error {
	sess, err := session.Get(SessionName, c)
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if userID == nil {
		return c.NoContent(http.StatusInternalServerError)
	}

	var req AddCourseRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bind request: %s", err))
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	courseID := uuid.NewRandom()
	_, err = tx.Exec("INSERT INTO `courses` (`id`, `code`, `type`, `name`, `description`, `credit`, `period`, `day_of_week`, `teacher_id`, `keywords`, `created_at`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())",
		courseID, req.Code, req.Type, req.Name, req.Description, req.Credit, req.Period, req.DayOfWeek, userID, req.Keywords)
	if err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	announcementID := uuid.NewRandom()
	_, err = tx.Exec("INSERT INTO `announcements` (`id`, `course_id`, `title`, `message`, `created_at`) VALUES (?, ?, ?, ?, NOW())",
		announcementID, courseID, fmt.Sprintf("コース追加: %s", req.Name), fmt.Sprintf("コースが新しく追加されました: %s\n%s", req.Name, req.Description))
	if err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	var users []*User
	if err := tx.Select(&users, "SELECT * FROM `users` WHERE `type` = ?", Student); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return c.NoContent(http.StatusInternalServerError)
	}

	// MEMO: N+1だけど最初から無くても良いかもしれない
	for _, user := range users {
		_, err := tx.Exec("INSERT INTO `unread_announcements` (`announcement_id`, `user_id`, `created_at`) VALUES (?, ?, NOW())",
			announcementID, user.ID)
		if err != nil {
			c.Logger().Error(err)
			_ = tx.Rollback()
			return c.NoContent(http.StatusInternalServerError)
		}
	}

	if err := tx.Commit(); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.JSON(http.StatusCreated, AddCourseResponse{ID: courseID})
}

type SetCourseStatusRequest struct {
	Status string `json:"status"`
}

func (h *handlers) SetCourseStatus(c echo.Context) error {
	courseID := uuid.Parse(c.Param("courseID"))
	if courseID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid courseID")
	}

	var req SetCourseStatusRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bind request: %s", err))
	}

	if _, err := h.DB.Exec("UPDATE `courses` SET `status` = ? WHERE `id` = ?", req.Status, courseID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	return c.NoContent(http.StatusOK)
}

type GetClassResponse struct {
	ID          uuid.UUID `json:"id"`
	Part        uint8     `json:"part"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
}

func (h *handlers) GetClasses(c echo.Context) error {
	courseID := uuid.Parse(c.Param("courseID"))
	if courseID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid courseID")
	}

	var course Course
	if err := h.DB.Get(&course, "SELECT * FROM `courses` WHERE `id` = ?", courseID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "course not found")
	} else if err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	var classes []Class
	if err := h.DB.Select(&classes, "SELECT * FROM `classes` WHERE `course_id` = ? ORDER BY `part`", courseID); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	res := make([]GetClassResponse, 0, len(classes))
	for _, class := range classes {
		res = append(res, GetClassResponse{
			ID:          class.ID,
			Part:        class.Part,
			Title:       class.Title,
			Description: class.Description,
		})
	}

	return c.JSON(http.StatusOK, res)
}

func (h *handlers) SubmitAssignment(context echo.Context) error {
	sess, err := session.Get(SessionName, context)
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if userID == nil {
		return context.NoContent(http.StatusInternalServerError)
	}

	courseID := uuid.Parse(context.Param("courseID"))
	if courseID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid courseID")
	}

	if ok, err := h.courseIsInCurrentPhase(courseID); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	} else if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "The course is not started yet or has ended.")
	}

	assignmentID := context.Param("assignmentID")
	var assignments int
	if err := h.DB.Get(&assignments, "SELECT COUNT(*) FROM `assignments` WHERE `id` = ?", assignmentID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusBadRequest, "No such assignment.")
	} else if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	file, err := context.FormFile("file")
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	src, err := file.Open()
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	defer src.Close()

	submissionID := uuid.New()
	dst, err := os.Create(AssignmentsDirectory + submissionID)
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	if _, err := h.DB.Exec("INSERT INTO `submissions` (`id`, `user_id`, `assignment_id`, `name`, `created_at`) VALUES (?, ?, ?, ?, NOW(6))", submissionID, userID, assignmentID, file.Filename); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	return context.NoContent(http.StatusNoContent)
}

func (h *handlers) DownloadSubmittedAssignment(c echo.Context) error {
	courseID := uuid.Parse(c.Param("courseID"))
	if courseID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid courseID")
	}

	assignmentID := uuid.Parse(c.Param("assignmentID"))
	if assignmentID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid assignmentID")
	}

	tx, err := h.DB.Beginx()
	if err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// MEMO: zipファイルを作るためFOR UPDATEでassignment、FOR SHAREでsubmissionをロック
	var assignment Assignment
	if err := tx.Get(&assignment, "SELECT * FROM `assignments` WHERE `id` = ? FOR UPDATE", assignmentID); err == sql.ErrNoRows {
		_ = tx.Rollback()
		return echo.NewHTTPError(http.StatusBadRequest, "No such assignment.")
	} else if err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	var submissions []*SubmissionWithUserName
	if err := tx.Select(&submissions,
		"SELECT `submissions`.*, `users`.`name` AS `user_name` "+
			"FROM `submissions` JOIN `users` ON `users`.`id` = `submissions`.`user_id`"+
			"WHERE `assignment_id` = ? ORDER BY `user_id` FOR SHARE", assignmentID); err != nil && err != sql.ErrNoRows {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	// MEMO: TODO: export時でなく提出時にzipファイルを作ることでボトルネックを作りたいが、「そうはならんやろ」という気持ち
	zipFilePath := AssignmentTmpDirectory + assignmentID.String() + ".zip"
	if err := createSubmissionsZip(zipFilePath, submissions); err != nil {
		c.Logger().Error(err)
		_ = tx.Rollback()
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	if err := tx.Commit(); err != nil {
		c.Logger().Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError)
	}

	return c.File(zipFilePath)
}

type AddClassRequest struct {
	Part        uint8  `json:"part"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type AddClassResponse struct {
	ID uuid.UUID `json:"id"`
}

func (h *handlers) AddClass(c echo.Context) error {
	courseID := uuid.Parse(c.Param("courseID"))
	if courseID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid courseID")
	}

	var req AddClassRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bind request: %v", err))
	}

	classID := uuid.NewRandom()
	if _, err := h.DB.Exec("INSERT INTO `classes` (`id`, `course_id`, `part`, `title`, `description`, `created_at`) VALUES (?, ?, ?, ?, ?, NOW(6))",
		classID, courseID, req.Part, req.Title, req.Description); err != nil {
		c.Logger().Error(err)
		return c.NoContent(http.StatusInternalServerError)
	}

	res := AddClassResponse{
		ID: classID,
	}

	return c.JSON(http.StatusCreated, res)
}

func (h *handlers) AddAnnouncements(context echo.Context) error {
	courseID := uuid.Parse(context.Param("courseID"))
	if courseID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid courseID")
	}
	var count int
	if err := h.DB.Get(&count, "SELECT COUNT(*) FROM `courses` WHERE `id` = ?", courseID); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	if count == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "No such course.")
	}

	var req PostAnnouncementsRequest
	if err := context.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("bind request: %v", err))
	}

	announcementID := uuid.NewRandom()
	if _, err := h.DB.Exec("INSERT INTO `announcements` (`id`, `course_id`, `title`, `message`, `created_at`) VALUES (?, ?, ?, ?, NOW(6))", announcementID, courseID, req.Title, req.Message); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	res := PostAnnouncementsResponse{
		ID: announcementID,
	}

	return context.JSON(http.StatusCreated, res)
}

func (h *handlers) GetAnnouncementList(context echo.Context) error {
	sess, err := session.Get(SessionName, context)
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if userID == nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	// MEMO: ページングの初期実装はページ番号形式
	var page int
	if context.QueryParam("page") == "" {
		page = 1
	} else {
		page, err = strconv.Atoi(context.QueryParam("page"))
		if err != nil || page <= 0 {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid page.")
		}
	}
	limit := 20
	offset := limit * (page - 1)

	announcements := make([]Announcement, 0)
	if err := h.DB.Select(&announcements, "SELECT `announcements`.`id`, `courses`.`name`, `announcements`.`title`, `announcements`.`message`, `announcements`.`created_at` "+
		"FROM `announcements` "+
		"JOIN `courses` ON `announcements`.`course_id` = `courses`.`id` "+
		"JOIN `registrations` ON `announcements`.`course_id` = `registrations`.`course_id` "+
		"WHERE `registrations`.`user_id` = ? AND `registrations`.`deleted_at` IS NULL "+
		"ORDER BY `announcements`.`created_at` DESC "+
		"LIMIT ? OFFSET ?", userID, limit+1, offset); err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	lenRes := len(announcements)
	if len(announcements) == limit+1 {
		lenRes = limit
	}
	res := make(GetAnnouncementsResponse, 0, lenRes)
	for _, announcement := range announcements[:lenRes] {
		res = append(res, GetAnnouncementResponse{
			ID:         announcement.ID,
			CourseName: announcement.CourseName,
			Title:      announcement.Title,
			CreatedAt:  announcement.CreatedAt.UnixNano() / int64(time.Millisecond),
		})
	}

	if lenRes > 0 {
		var links []string
		path := fmt.Sprintf("%v://%v%v", context.Scheme(), context.Request().Host, context.Path())
		if page > 1 {
			links = append(links, fmt.Sprintf("<%v?page=%v>; rel=\"prev\"", path, page-1))
		}
		if len(announcements) == limit+1 {
			links = append(links, fmt.Sprintf("<%v?page=%v>; rel=\"next\"", path, page+1))
		}
		context.Response().Header().Set("Link", strings.Join(links, ","))
	}

	return context.JSON(http.StatusOK, res)
}

func (h *handlers) GetAnnouncementDetail(context echo.Context) error {
	sess, err := session.Get(SessionName, context)
	if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}
	userID := uuid.Parse(sess.Values["userID"].(string))
	if userID == nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	announcementID := uuid.Parse(context.Param("announcementID"))
	if announcementID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid announcementID")
	}

	var announcement Announcement
	if err := h.DB.Get(&announcement, "SELECT `announcements`.`id`, `courses`.`name`, `announcements`.`title`, `announcements`.`message`, `announcements`.`created_at`"+
		"FROM `announcements`"+
		"JOIN `courses` ON `announcements`.`course_id` = `courses`.`id`"+
		"JOIN `registrations` ON `announcements`.`course_id` = `registrations`.`course_id`"+
		"WHERE `announcements`.`id` = ? AND `registrations`.`user_id` = ? AND `registrations`.`deleted_at` IS NULL", announcementID, userID); err == sql.ErrNoRows {
		return echo.NewHTTPError(http.StatusNotFound, "announcement not found.")
	} else if err != nil {
		log.Println(err)
		return context.NoContent(http.StatusInternalServerError)
	}

	res := GetAnnouncementDetailResponse{
		ID:         announcement.ID,
		CourseName: announcement.CourseName,
		Title:      announcement.Title,
		Message:    announcement.Message,
		CreatedAt:  announcement.CreatedAt.UnixNano() / int64(time.Millisecond),
	}
	return context.JSON(http.StatusOK, res)
}

func (h *handlers) courseIsInCurrentPhase(courseID uuid.UUID) (bool, error) {
	// MEMO: 複数phaseに渡る講義を想定していない
	var schedule Schedule
	query := "SELECT `schedules`.*" +
		"FROM `schedules`" +
		"JOIN `course_schedules` ON `schedules`.`id` = `course_schedules`.`schedule_id`" +
		"JOIN `courses` ON `course_schedules`.`course_id` = `courses`.`id`" +
		"WHERE `courses`.`id` = ?" +
		"LIMIT 1"
	if err := h.DB.Get(&schedule, query, courseID); err != nil {
		return false, err
	}

	var phase Phase
	if err := h.DB.Get(&phase, "SELECT * FROM `phase`"); err != nil {
		return false, err
	}

	return schedule.Year == phase.Year && schedule.Semester == phase.Semester, nil
}

func createSubmissionsZip(zipFilePath string, submissions []*SubmissionWithUserName) error {
	// Zipに含めるファイルの名称変更のためコピー
	// MEMO: N回 cp はやりすぎかも
	for _, submission := range submissions {
		cpCmd := exec.Command(
			"cp",
			AssignmentsDirectory+submission.ID.String(),
			AssignmentTmpDirectory+submission.UserName+"-"+submission.ID.String()+"-"+submission.Name,
		)
		if err := cpCmd.Start(); err != nil {
			return err
		}
		if err := cpCmd.Wait(); err != nil {
			return err
		}
	}

	zipArgs := make([]string, 0, len(submissions)+2)
	zipArgs = append(zipArgs, "-j", zipFilePath)
	for _, submission := range submissions {
		zipArgs = append(zipArgs, AssignmentTmpDirectory+submission.UserName+"-"+submission.ID.String()+"-"+submission.Name)
	}
	cmd := exec.Command("zip", zipArgs...)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Wait()
}
