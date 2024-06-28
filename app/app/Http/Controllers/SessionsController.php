<?php

namespace App\Http\Controllers;

use App\Enums\TestResultEnum;

class SessionsController extends Controller
{
    private $sessions;

    private $suites;

    public function __construct()
    {
        $this->sessions = [
            (object) [
                'id' => 1,
                'title' => 'Real Rewards API v3.6',
                'timestamp' => '2024-05-23 02:30:05',
                'issuer' => 'Norbert Laposa',
                'runs' => [
                    (object) [
                        'id' => 1,
                        'timestamp' => now(),
                        'passed' => 145,
                        'failed' => 5,
                        'result_log' => 'json data from GitHub Actions API',
                    ],
                ],
                'suites' => [
                    (object) [
                        'id' => 1,
                        'title' => 'musgrave-supervalu/loyalty-mobile-app/01_Identity_Login',
                        'url' =>
                            'https://github.com/laposa/musgrave-supervalu/tree/master/tests/loyalty-mobile-app/maestro/.flows/automated/01_Identity_Login',
                        'passed' => 4,
                        'failed' => 2,
                        'tests' => [
                            (object) [
                                'id' => '001_User_can_login',
                                'title' => '001_User_can_login',
                                'type' => 'automated',
                                'url' =>
                                    'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/01_Identity_Login/001_User_can_login.yaml',
                            ],
                            (object) [
                                'id' => '002_User_cannot_login_with_invalid_credentials',
                                'title' => '002_User_cannot_login_with_invalid_credentials',
                                'type' => 'automated',
                                'url' =>
                                    'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/01_Identity_Login/002_User_cannot_login_with_invalid_credentials.yaml',
                            ],
                        ],
                    ],
                    (object) [
                        'id' => 2,
                        'title' => 'musgrave-supervalu/loyalty-mobile-app/02_Identity_Registration',
                        'url' =>
                            'https://github.com/laposa/musgrave-supervalu/tree/master/tests/loyalty-mobile-app/maestro/.flows/automated/02_Identity_Registration',
                        'passed' => 4,
                        'failed' => 2,
                        'tests' => [
                            (object) [
                                'id' => '010_User_can_register_a_new_account_creating_a_new_card',
                                'title' =>
                                    '010_User_can_register_a_new_account_creating_a_new_card',
                                'type' => 'automated',
                                'url' =>
                                    'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/02_Identity_Registration/010_User_can_register_a_new_account_creating_a_new_card.yaml',
                            ],
                            (object) [
                                'id' => '219_User_has_valid_Register_screens_content_displayed',
                                'title' => '219_User_has_valid_Register_screens_content_displayed.',
                                'type' => 'automated',
                                'url' =>
                                    'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/02_Identity_Registration/219_User_has_valid_Register_screens_content_displayed.yaml',
                            ],
                        ],
                    ],
                ],
            ],
            (object) [
                'id' => 2,
                'title' => 'SuperValu Website security 05',
                'timestamp' => '2024-05-21 02:00:00',
                'issuer' => 'Hugo Dvorak',
                'runs' => [
                    (object) [
                        'id' => 1,
                        'timestamp' => now(),
                        'result' => TestResultEnum::pass->value,
                        'suite' => 'Suite 1',
                        'session' => 'Session 1',
                        'code' => '',
                        'error' => '',
                        'test' => (object) [
                            'title' => 'Login screen renders',
                        ],
                    ],
                ],
                'suites' => [
                    (object) [
                        'id' => 1,
                        'title' => 'musgrave-supervalu/loyalty-mobile-app/01_Identity_Login',
                        'url' =>
                            'https://github.com/laposa/musgrave-supervalu/tree/master/tests/loyalty-mobile-app/maestro/.flows/automated/01_Identity_Login',
                        'passed' => 4,
                        'failed' => 2,
                        'tests' => [
                            (object) [
                                'id' => '001_User_can_login',
                                'title' => '001_User_can_login',
                                'type' => 'automated',
                                'url' =>
                                    'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/01_Identity_Login/001_User_can_login.yaml',
                            ],
                            (object) [
                                'id' => '002_User_cannot_login_with_invalid_credentials',
                                'title' => '002_User_cannot_login_with_invalid_credentials',
                                'type' => 'automated',
                                'url' =>
                                    'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/01_Identity_Login/002_User_cannot_login_with_invalid_credentials.yaml',
                            ],
                        ],
                    ],
                    (object) [
                        'id' => 2,
                        'title' => 'musgrave-supervalu/loyalty-mobile-app/02_Identity_Registration',
                        'url' =>
                            'https://github.com/laposa/musgrave-supervalu/tree/master/tests/loyalty-mobile-app/maestro/.flows/automated/02_Identity_Registration',
                        'passed' => 4,
                        'failed' => 2,
                        'tests' => [
                            (object) [
                                'id' => '010_User_can_register_a_new_account_creating_a_new_card',
                                'title' =>
                                    '010_User_can_register_a_new_account_creating_a_new_card',
                                'type' => 'automated',
                                'url' =>
                                    'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/02_Identity_Registration/010_User_can_register_a_new_account_creating_a_new_card.yaml',
                            ],
                            (object) [
                                'id' => '219_User_has_valid_Register_screens_content_displayed',
                                'title' => '219_User_has_valid_Register_screens_content_displayed.',
                                'type' => 'automated',
                                'url' =>
                                    'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/02_Identity_Registration/219_User_has_valid_Register_screens_content_displayed.yaml',
                            ],
                        ],
                    ],
                ],
            ],
            (object) [
                'id' => 3,
                'title' => 'Frank and Honest v1.3',
                'timestamp' => '2024-04-08 00:00:01',
                'issuer' => 'Martin Miksovsky',
                'runs' => [
                    (object) [
                        'id' => 1,
                        'timestamp' => now(),
                        'result' => TestResultEnum::pass->value,
                        'suite' => 'Suite 1',
                        'session' => 'Session 1',
                        'code' => '',
                        'error' => '',
                        'test' => (object) [
                            'title' => 'Login screen renders',
                        ],
                    ],
                ],
                'suites' => [
                    (object) [
                        'id' => 1,
                        'title' => 'musgrave-supervalu/loyalty-mobile-app/01_Identity_Login',
                        'url' =>
                            'https://github.com/laposa/musgrave-supervalu/tree/master/tests/loyalty-mobile-app/maestro/.flows/automated/01_Identity_Login',
                        'passed' => 4,
                        'failed' => 2,
                        'tests' => [
                            (object) [
                                'id' => '001_User_can_login',
                                'title' => '001_User_can_login',
                                'type' => 'automated',
                                'url' =>
                                    'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/01_Identity_Login/001_User_can_login.yaml',
                            ],
                            (object) [
                                'id' => '002_User_cannot_login_with_invalid_credentials',
                                'title' => '002_User_cannot_login_with_invalid_credentials',
                                'type' => 'automated',
                                'url' =>
                                    'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/01_Identity_Login/002_User_cannot_login_with_invalid_credentials.yaml',
                            ],
                        ],
                    ],
                    (object) [
                        'id' => 2,
                        'title' => 'musgrave-supervalu/loyalty-mobile-app/02_Identity_Registration',
                        'url' =>
                            'https://github.com/laposa/musgrave-supervalu/tree/master/tests/loyalty-mobile-app/maestro/.flows/automated/02_Identity_Registration',
                        'passed' => 4,
                        'failed' => 2,
                        'tests' => [
                            (object) [
                                'id' => '010_User_can_register_a_new_account_creating_a_new_card',
                                'title' =>
                                    '010_User_can_register_a_new_account_creating_a_new_card',
                                'type' => 'automated',
                                'url' =>
                                    'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/02_Identity_Registration/010_User_can_register_a_new_account_creating_a_new_card.yaml',
                            ],
                            (object) [
                                'id' => '219_User_has_valid_Register_screens_content_displayed',
                                'title' => '219_User_has_valid_Register_screens_content_displayed.',
                                'type' => 'automated',
                                'url' =>
                                    'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/02_Identity_Registration/219_User_has_valid_Register_screens_content_displayed.yaml',
                            ],
                        ],
                    ],
                ],
            ],
            (object) [
                'id' => 4,
                'title' => 'SuperValu Website security 04',
                'timestamp' => '2024-04-25 00:00:01',
                'issuer' => 'Martin Miksovsky',
                'runs' => [
                    (object) [
                        'id' => 1,
                        'timestamp' => now(),
                        'result' => TestResultEnum::pass->value,
                        'suite' => 'Suite 1',
                        'session' => 'Session 1',
                        'code' => '',
                        'error' => '',
                        'test' => (object) [
                            'title' => 'Login screen renders',
                        ],
                    ],
                ],
                'suites' => [
                    (object) [
                        'id' => 1,
                        'title' => 'musgrave-supervalu/loyalty-mobile-app/01_Identity_Login',
                        'url' =>
                            'https://github.com/laposa/musgrave-supervalu/tree/master/tests/loyalty-mobile-app/maestro/.flows/automated/01_Identity_Login',
                        'passed' => 4,
                        'failed' => 2,
                        'tests' => [
                            (object) [
                                'id' => '001_User_can_login',
                                'title' => '001_User_can_login',
                                'type' => 'automated',
                                'url' =>
                                    'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/01_Identity_Login/001_User_can_login.yaml',
                            ],
                            (object) [
                                'id' => '002_User_cannot_login_with_invalid_credentials',
                                'title' => '002_User_cannot_login_with_invalid_credentials',
                                'type' => 'automated',
                                'url' =>
                                    'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/01_Identity_Login/002_User_cannot_login_with_invalid_credentials.yaml',
                            ],
                        ],
                    ],
                    (object) [
                        'id' => 2,
                        'title' => 'musgrave-supervalu/loyalty-mobile-app/02_Identity_Registration',
                        'url' =>
                            'https://github.com/laposa/musgrave-supervalu/tree/master/tests/loyalty-mobile-app/maestro/.flows/automated/02_Identity_Registration',
                        'passed' => 4,
                        'failed' => 2,
                        'tests' => [
                            (object) [
                                'id' => '010_User_can_register_a_new_account_creating_a_new_card',
                                'title' =>
                                    '010_User_can_register_a_new_account_creating_a_new_card',
                                'type' => 'automated',
                                'url' =>
                                    'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/02_Identity_Registration/010_User_can_register_a_new_account_creating_a_new_card.yaml',
                            ],
                            (object) [
                                'id' => '219_User_has_valid_Register_screens_content_displayed',
                                'title' => '219_User_has_valid_Register_screens_content_displayed.',
                                'type' => 'automated',
                                'url' =>
                                    'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/02_Identity_Registration/219_User_has_valid_Register_screens_content_displayed.yaml',
                            ],
                        ],
                    ],
                ],
            ],
        ];

        $this->suites = [
            (object) [
                'id' => '01_Identity_Login',
                'title' => 'musgrave-supervalu/loyalty-mobile-app/01_Identity_Login',
                'url' =>
                    'https://github.com/laposa/musgrave-supervalu/tree/master/tests/loyalty-mobile-app/maestro/.flows/automated/01_Identity_Login',
                'tests' => [
                    (object) [
                        'id' => '001_User_can_login',
                        'title' => '001_User_can_login',
                        'type' => 'automated',
                        'url' =>
                            'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/01_Identity_Login/001_User_can_login.yaml',
                    ],
                    (object) [
                        'id' => '002_User_cannot_login_with_invalid_credentials',
                        'title' => '002_User_cannot_login_with_invalid_credentials',
                        'type' => 'automated',
                        'url' =>
                            'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/01_Identity_Login/002_User_cannot_login_with_invalid_credentials.yaml',
                    ],
                ],
            ],
            (object) [
                'id' => '02_Identity_Registration',
                'title' => 'musgrave-supervalu/loyalty-mobile-app/02_Identity_Registration',
                'url' =>
                    'https://github.com/laposa/musgrave-supervalu/tree/master/tests/loyalty-mobile-app/maestro/.flows/automated/02_Identity_Registration',
                'tests' => [
                    (object) [
                        'id' => '010_User_can_register_a_new_account_creating_a_new_card',
                        'title' => '010_User_can_register_a_new_account_creating_a_new_card',
                        'type' => 'automated',
                        'url' =>
                            'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/02_Identity_Registration/010_User_can_register_a_new_account_creating_a_new_card.yaml',
                    ],
                    (object) [
                        'id' => '219_User_has_valid_Register_screens_content_displayed',
                        'title' => '219_User_has_valid_Register_screens_content_displayed.',
                        'type' => 'automated',
                        'url' =>
                            'https://github.com/laposa/musgrave-supervalu/blob/master/tests/loyalty-mobile-app/maestro/.flows/automated/02_Identity_Registration/219_User_has_valid_Register_screens_content_displayed.yaml',
                    ],
                ],
            ],
        ];
    }

    public function index()
    {
        return view('sessions.index', [
            'sessions' => $this->sessions,
        ]);
    }

    public function show()
    {
        return view('sessions.show', [
            'session' => $this->sessions[1],
        ]);
    }

    public function create()
    {
        return view('sessions.create', [
            'suites' => $this->suites,
        ]);
    }

    public function store()
    {
        return view('sessions.show');
    }
}
