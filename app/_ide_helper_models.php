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
 * @property string $commentable_type
 * @property int $commentable_id
 * @property int $user_id
 * @property string $body
 * @property \Illuminate\Support\Carbon|null $created_at
 * @property \Illuminate\Support\Carbon|null $updated_at
 * @property-read \Illuminate\Database\Eloquent\Model|\Eloquent $commentable
 * @property-read \App\Models\User $user
 * @method static \Database\Factories\CommentFactory factory($count = null, $state = [])
 * @method static \Illuminate\Database\Eloquent\Builder|Comment newModelQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|Comment newQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|Comment query()
 * @method static \Illuminate\Database\Eloquent\Builder|Comment whereBody($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Comment whereCommentableId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Comment whereCommentableType($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Comment whereCreatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Comment whereId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Comment whereUpdatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|Comment whereUserId($value)
 */
	class Comment extends \Eloquent {}
}

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
 * @property string $reviewable_type
 * @property int $reviewable_id
 * @property int $requester_id
 * @property int $reviewer_id
 * @property string $status
 * @property string|null $completed_at
 * @property \Illuminate\Support\Carbon|null $created_at
 * @property \Illuminate\Support\Carbon|null $updated_at
 * @property-read \App\Models\User $requester
 * @property-read \Illuminate\Database\Eloquent\Model|\Eloquent $reviewable
 * @property-read \App\Models\User $reviewer
 * @method static \Database\Factories\ReviewRequestFactory factory($count = null, $state = [])
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest newModelQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest newQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest query()
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest whereCompletedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest whereCreatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest whereId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest whereRequesterId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest whereReviewableId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest whereReviewableType($value)
 * @method static \Illuminate\Database\Eloquent\Builder|ReviewRequest whereReviewerId($value)
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
 * @property-read \App\Models\Installation $installation
 * @property-read \App\Models\User $issuer
 * @property-read \Illuminate\Database\Eloquent\Collection<int, \App\Models\SessionItem> $items
 * @property-read int|null $items_count
 * @property-read mixed $items_grouped
 * @property-read mixed $items_grouped_by_service
 * @property-read \Illuminate\Database\Eloquent\Collection<int, \App\Models\SessionRun> $runs
 * @property-read int|null $runs_count
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
 * @property int $repository_id
 * @property int $workflow_id
 * @property string $repository_name
 * @property string $service_name
 * @property string $suite_name
 * @property string $test_name
 * @property string $service_url
 * @property \Illuminate\Support\Carbon|null $created_at
 * @property \Illuminate\Support\Carbon|null $updated_at
 * @property-read \Illuminate\Database\Eloquent\Collection<int, \App\Models\SessionRun> $runs
 * @property-read int|null $runs_count
 * @property-read \App\Models\Session $session
 * @method static \Illuminate\Database\Eloquent\Builder|SessionItem newModelQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionItem newQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionItem query()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionItem whereCreatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionItem whereId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionItem whereRepositoryId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionItem whereRepositoryName($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionItem whereServiceName($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionItem whereServiceUrl($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionItem whereSessionId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionItem whereSuiteName($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionItem whereTestName($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionItem whereUpdatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionItem whereWorkflowId($value)
 */
	class SessionItem extends \Eloquent {}
}

namespace App\Models{
/**
 * 
 *
 * @property int $id
 * @property int $session_id
 * @property string $status
 * @property int|null $passed
 * @property int|null $failed
 * @property string|null $result_log
 * @property string|null $run_log
 * @property \Illuminate\Support\Carbon|null $created_at
 * @property \Illuminate\Support\Carbon|null $updated_at
 * @property-read \Illuminate\Database\Eloquent\Collection<int, \App\Models\Comment> $comments
 * @property-read int|null $comments_count
 * @property-read \Illuminate\Database\Eloquent\Collection<int, \App\Models\SessionItem> $items
 * @property-read int|null $items_count
 * @property-read mixed $items_grouped
 * @property-read mixed $parsed_results
 * @property-read \Illuminate\Database\Eloquent\Collection<int, \App\Models\ReviewRequest> $reviewRequests
 * @property-read int|null $review_requests_count
 * @property-read \App\Models\Session $session
 * @method static \Illuminate\Database\Eloquent\Builder|SessionRun newModelQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionRun newQuery()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionRun query()
 * @method static \Illuminate\Database\Eloquent\Builder|SessionRun whereCreatedAt($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionRun whereFailed($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionRun whereId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionRun wherePassed($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionRun whereResultLog($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionRun whereRunLog($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionRun whereSessionId($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionRun whereStatus($value)
 * @method static \Illuminate\Database\Eloquent\Builder|SessionRun whereUpdatedAt($value)
 */
	class SessionRun extends \Eloquent {}
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

