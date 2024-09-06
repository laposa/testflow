<?php

// @formatter:off
// phpcs:ignoreFile
/**
 * A helper file for your Eloquent Models
 * Copy the phpDocs from this file to the correct Model,
 * And remove them from this file, to prevent double declarations.
 *
 * @author Barry vd. Heuvel <barryvdh@gmail.com>
 */


namespace App\Models{
/**
 * 
 *
 * @property int $id
 * @property string $installation_id
 * @property string $access_token
 * @property string $expires_at
 * @property string $repository_selection
 * @property \Illuminate\Support\Carbon|null $created_at
 * @property \Illuminate\Support\Carbon|null $updated_at
 * @property-read \Illuminate\Database\Eloquent\Collection<int, \App\Models\Session> $sessions
 * @property-read int|null $sessions_count
 * @method static \Illuminate\Database\Eloquent\Builder|Installation newModelQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|Installation newQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|Installation query()
 * @method static \Illuminate\Database\Eloquent\Builder|Installation whereAccessToken($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Installation whereCreatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Installation whereExpiresAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Installation whereId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Installation whereInstallationId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Installation whereRepositorySelection($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Installation whereUpdatedAt($value)
 */
	class Installation extends \Eloquent {}
}

namespace App\Models{
/**
 * 
 *
 * @property int $id
 * @property int $session_id
 * @property int $requester_id
 * @property int $reviewer_id
 * @property string $status
 * @property string|null $completed_at
 * @property \Illuminate\Support\Carbon|null $created_at
 * @property \Illuminate\Support\Carbon|null $updated_at
 * @property-read \App\Models\User $requester
 * @property-read \App\Models\User $reviewer
 * @property-read \App\Models\Session $session
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest newModelQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest newQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest query()
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest whereCompletedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest whereCreatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest whereId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest whereRequesterId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest whereReviewerId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest whereSessionId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest whereStatus($value)
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest whereUpdatedAt($value)
 */
	class ReviewRequest extends \Eloquent {}
}

namespace App\Models{
/**
 * 
 *
 * @property int $id
 * @property int $installation_id
 * @property int $issuer_id
 * @property string $name
 * @property string $environment
 * @property \Illuminate\Support\Carbon|null $created_at
 * @property \Illuminate\Support\Carbon|null $updated_at
 * @property-read \Illuminate\Database\Eloquent\Collection<int, \App\Models\SessionActivity> $activity
 * @property-read int|null $activity_count
 * @property-read \App\Models\Installation $installation
 * @property-read \App\Models\User $issuer
 * @property-read \Illuminate\Database\Eloquent\Collection<int, \App\Models\SessionServiceRun> $lastRuns
 * @property-read int|null $last_runs_count
 * @property-read \Illuminate\Database\Eloquent\Collection<int, \App\Models\ReviewRequest> $reviewRequests
 * @property-read int|null $review_requests_count
 * @property-read \Illuminate\Database\Eloquent\Collection<int, \App\Models\SessionServiceRun> $runs
 * @property-read int|null $runs_count
 * @property-read \Illuminate\Database\Eloquent\Collection<int, \App\Models\SessionService> $services
 * @property-read int|null $services_count
 * @method static \Illuminate\Database\Eloquent\Builder|Session newModelQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|Session newQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|Session query()
 * @method static \Illuminate\Database\Eloquent\Builder|Session whereCreatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Session whereEnvironment($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Session whereId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Session whereInstallationId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Session whereIssuerId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Session whereName($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Session whereUpdatedAt($value)
 */
	class Session extends \Eloquent {}
}

namespace App\Models{
/**
 * 
 *
 * @property int $id
 * @property int $session_id
 * @property int $user_id
 * @property \App\Enums\SessionActivityType $type
 * @property string|null $body
 * @property array|null $metadata
 * @property \Illuminate\Support\Carbon|null $created_at
 * @property \Illuminate\Support\Carbon|null $updated_at
 * @property-read \App\Models\Session $session
 * @property-read \App\Models\User $user
 * @method static \Illuminate\Database\Eloquent\Builder|SessionActivity newModelQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionActivity newQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionActivity query()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionActivity whereBody($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionActivity whereCreatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionActivity whereId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionActivity whereMetadata($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionActivity whereSessionId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionActivity whereType($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionActivity whereUpdatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionActivity whereUserId($value)
 */
	class SessionActivity extends \Eloquent {}
}

namespace App\Models{
/**
 * 
 *
 * @property int $id
 * @property int $session_id
 * @property int $repository_id
 * @property int $workflow_id
 * @property string $name
 * @property string $url
 * @property string $repository_name
 * @property \Illuminate\Support\Carbon|null $created_at
 * @property \Illuminate\Support\Carbon|null $updated_at
 * @property-read mixed $display_name
 * @property-read mixed $repository_name_without_owner
 * @property-read \Illuminate\Database\Eloquent\Collection<int, \App\Models\SessionServiceRun> $runs
 * @property-read int|null $runs_count
 * @property-read \App\Models\Session $session
 * @property-read \Illuminate\Database\Eloquent\Collection<int, \App\Models\SessionServiceSuite> $suites
 * @property-read int|null $suites_count
 * @method static \Illuminate\Database\Eloquent\Builder|SessionService newModelQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionService newQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionService query()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionService whereCreatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionService whereId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionService whereName($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionService whereRepositoryId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionService whereRepositoryName($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionService whereSessionId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionService whereUpdatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionService whereUrl($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionService whereWorkflowId($value)
 */
	class SessionService extends \Eloquent {}
}

namespace App\Models{
/**
 * 
 *
 * @property int $id
 * @property int $service_id
 * @property string $status
 * @property int|null $passed
 * @property int|null $failed
 * @property int|null $skipped
 * @property int|null $duration
 * @property string|null $result_log
 * @property string|null $run_log
 * @property \Illuminate\Support\Carbon|null $created_at
 * @property \Illuminate\Support\Carbon|null $updated_at
 * @property-read mixed $human_readable_duration
 * @property-read mixed $parsed_results
 * @property-read \App\Models\SessionService $service
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceRun newModelQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceRun newQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceRun query()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceRun whereCreatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceRun whereDuration($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceRun whereFailed($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceRun whereId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceRun wherePassed($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceRun whereResultLog($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceRun whereRunLog($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceRun whereServiceId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceRun whereSkipped($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceRun whereStatus($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceRun whereUpdatedAt($value)
 */
	class SessionServiceRun extends \Eloquent {}
}

namespace App\Models{
/**
 * 
 *
 * @property int $id
 * @property int $service_id
 * @property string $name
 * @property string $url
 * @property \Illuminate\Support\Carbon|null $created_at
 * @property \Illuminate\Support\Carbon|null $updated_at
 * @property-read \App\Models\SessionService $service
 * @property-read \Illuminate\Database\Eloquent\Collection<int, \App\Models\SessionServiceSuiteTest> $tests
 * @property-read int|null $tests_count
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuite newModelQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuite newQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuite query()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuite whereCreatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuite whereId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuite whereName($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuite whereServiceId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuite whereUpdatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuite whereUrl($value)
 */
	class SessionServiceSuite extends \Eloquent {}
}

namespace App\Models{
/**
 * 
 *
 * @property int $id
 * @property int $suite_id
 * @property string $name
 * @property string $url
 * @property \Illuminate\Support\Carbon|null $created_at
 * @property \Illuminate\Support\Carbon|null $updated_at
 * @property-read \App\Models\SessionServiceSuite $suite
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuiteTest newModelQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuiteTest newQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuiteTest query()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuiteTest whereCreatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuiteTest whereId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuiteTest whereName($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuiteTest whereSuiteId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuiteTest whereUpdatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionServiceSuiteTest whereUrl($value)
 */
	class SessionServiceSuiteTest extends \Eloquent {}
}

namespace App\Models{
/**
 * 
 *
 * @property int $id
 * @property string $name
 * @property string $email
 * @property \Illuminate\Support\Carbon|null $email_verified_at
 * @property mixed|null $password
 * @property string|null $github_id
 * @property string|null $github_token
 * @property string|null $github_refresh_token
 * @property string|null $remember_token
 * @property \Illuminate\Support\Carbon|null $created_at
 * @property \Illuminate\Support\Carbon|null $updated_at
 * @property-read \Illuminate\Notifications\DatabaseNotificationCollection<int, \Illuminate\Notifications\DatabaseNotification> $notifications
 * @property-read int|null $notifications_count
 * @method static \Database\Factories\UserFactory factory($count = null, $state = [])
 * @method static \Illuminate\Database\Eloquent\Builder|User newModelQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|User newQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|User query()
 * @method static \Illuminate\Database\Eloquent\Builder|User whereCreatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|User whereEmail($value)
 * @method static \Illuminate\Database\Eloquent\Builder|User whereEmailVerifiedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|User whereGithubId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|User whereGithubRefreshToken($value)
 * @method static \Illuminate\Database\Eloquent\Builder|User whereGithubToken($value)
 * @method static \Illuminate\Database\Eloquent\Builder|User whereId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|User whereName($value)
 * @method static \Illuminate\Database\Eloquent\Builder|User wherePassword($value)
 * @method static \Illuminate\Database\Eloquent\Builder|User whereRememberToken($value)
 * @method static \Illuminate\Database\Eloquent\Builder|User whereUpdatedAt($value)
 */
	class User extends \Eloquent {}
}

