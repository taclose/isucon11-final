INSERT INTO `users` (`id`, `code`, `name`, `hashed_password`, `type`) VALUES
(
  '01234567-89ab-cdef-0001-000000000001',
  'S0000001',
  '佐藤太郎',
  '$2a$10$KEgha.chGu1/N4kHZ./rIeK1QISkv8sYk15Mqktr6BGB8xomRRe02', -- "password"
  'student'
),
(
  '01234567-89ab-cdef-0001-000000000002',
  'S0000002',
  '鈴木次郎',
  '$2a$10$KEgha.chGu1/N4kHZ./rIeK1QISkv8sYk15Mqktr6BGB8xomRRe02',
  'student'
),
(
  '01234567-89ab-cdef-0001-000000000003',
  'S0000003',
  '高橋三郎',
  '$2a$10$KEgha.chGu1/N4kHZ./rIeK1QISkv8sYk15Mqktr6BGB8xomRRe02',
  'student'
),
(
  '01234567-89ab-cdef-0001-000000000004',
  'F0000001',
  '椅子昆',
  '$2a$10$KEgha.chGu1/N4kHZ./rIeK1QISkv8sYk15Mqktr6BGB8xomRRe02',
  'teacher'
),
(
  '01234567-89ab-cdef-0001-000000000005',
  'F0000002',
  '田山勝蔵',
  '$2a$10$KEgha.chGu1/N4kHZ./rIeK1QISkv8sYk15Mqktr6BGB8xomRRe02',
  'teacher'
);

INSERT INTO `courses` (`id`, `code`, `type`, `name`, `description`, `credit`, `period`, `day_of_week`, `teacher_id`, `keywords`, `status`, `created_at`) VALUES
(
  '01234567-89ab-cdef-0002-000000000001',
  'ISU.F117',
  'major-subjects',
  '微分積分基礎',
  '微積分の基礎を学びます。',
  2,
  1,
  'monday',
  '01234567-89ab-cdef-0001-000000000004',
  '数学 微分 積分 基礎',
  'registration',
  NOW()
),
(
  '01234567-89ab-cdef-0002-000000000002',
  'ISU.M101',
  'major-subjects',
  '線形代数基礎',
  '線形代数の基礎を学びます。',
  2,
  2,
  'tuesday',
  '01234567-89ab-cdef-0001-000000000004',
  '数学 線形代数 基礎',
  'registration',
  NOW()
),
(
  '01234567-89ab-cdef-0002-000000000003',
  'CSP.B003',
  'major-subjects',
  'プログラミング',
  'プログラミングを学びます。',
  2,
  3,
  'wednesday',
  '01234567-89ab-cdef-0001-000000000005',
  '計算機科学 C言語',
  'registration',
  NOW()
),
(
  '01234567-89ab-cdef-0002-000000000004',
  'CSC.B103',
  'major-subjects',
  'プログラミング演習A',
  'プログラミングの演習を行います。',
  1,
  4,
  'wednesday',
  '01234567-89ab-cdef-0001-000000000005',
  '計算機科学 C言語 演習',
  'registration',
  NOW()
),
(
  '01234567-89ab-cdef-0002-000000000005',
  'CSC.B104',
  'major-subjects',
  'プログラミング演習B',
  'プログラミングの演習を行います。',
  1,
  4,
  'wednesday',
  '01234567-89ab-cdef-0001-000000000005',
  '計算機科学 C言語 演習',
  'registration',
  NOW()
),
(
  '01234567-89ab-cdef-0002-000000000006',
  'LAH.A100',
  'liberal-arts',
  'ISUCON基礎',
  'ISUCONの基礎を学びます。',
  2,
  1,
  'thursday',
  '01234567-89ab-cdef-0001-000000000005',
  'ISUCON パフォーマンスチューニング',
  'registration',
  NOW()
),
(
  '01234567-89ab-cdef-0002-000000000007',
  'LAH.B101',
  'liberal-arts',
  '英語第一',
  '英語を学びます。',
  1,
  2,
  'friday',
  '01234567-89ab-cdef-0001-000000000005',
  '英語 English',
  'closed',
  NOW()
),
(
  '01234567-89ab-cdef-0002-000000000008',
  'LAH.B102',
  'liberal-arts',
  '英語第二',
  '英会話を実践します。',
  1,
  2,
  'friday',
  '01234567-89ab-cdef-0001-000000000005',
  '英語 英会話 English',
  'in-progress',
  NOW()
);

INSERT INTO `classes` (`id`, `course_id`, `part`, `title`, `description`, `created_at`) VALUES
(
  '01234567-89ab-cdef-0004-000000000001',
  '01234567-89ab-cdef-0002-000000000001',
  1,
  '微分積分基礎第一回',
  '微分積分の導入',
  NOW()
),
(
  '01234567-89ab-cdef-0004-000000000002',
  '01234567-89ab-cdef-0002-000000000001',
  2,
  '微分積分基礎第二回',
  '微分(1)',
  NOW()
),
(
  '01234567-89ab-cdef-0004-000000000003',
  '01234567-89ab-cdef-0002-000000000002',
  1,
  '線形代数基礎第一回',
  '線形代数とは',
  NOW()
);

INSERT INTO `registrations` (course_id, user_id, created_at) VALUES
(
  '01234567-89ab-cdef-0002-000000000001',
  '01234567-89ab-cdef-0001-000000000001',
  NOW()
);

INSERT INTO `submissions` (id, user_id, class_id, file_name, score, created_at) VALUES
(
  '01234567-89ab-cdef-0005-000000000001',
  '01234567-89ab-cdef-0001-000000000001',
  '01234567-89ab-cdef-0004-000000000001',
  'test.pdf',
  NULL,
  NOW()
),
(
  '01234567-89ab-cdef-0005-000000000002',
  '01234567-89ab-cdef-0001-000000000001',
  '01234567-89ab-cdef-0004-000000000002',
  'test.pdf',
  80,
  NOW()
);
