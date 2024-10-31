<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Casts\Attribute;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Database\Eloquent\Relations\HasManyThrough;

class Session extends Model
{
    protected $fillable = ['issuer_id', 'environment', 'name'];

    protected $table = 'test_sessions';


    public function issuer(): BelongsTo
    {
        return $this->belongsTo(User::class, foreignKey: 'issuer_id');
    }

    public function services(): HasMany
    {
        return $this->hasMany(SessionService::class);
    }

    public function runs(): HasManyThrough
    {
        return $this->hasManyThrough(
            SessionServiceRun::class,
            SessionService::class,
            'session_id',
            'service_id',
        )->select([
            'test_session_service_runs.id',
            'test_session_service_runs.service_id',
            'test_session_service_runs.status',
            'test_session_service_runs.passed',
            'test_session_service_runs.failed',
            'test_session_service_runs.skipped',
            'test_session_service_runs.duration',
            'test_session_service_runs.created_at',
            'test_session_service_runs.started_at',
            'test_session_service_runs.finished_at',
            'test_session_service_runs.result_log',
        ]);
    }

    public function lastRuns(): HasManyThrough
    {
        return $this->runs()->whereIn('test_session_service_runs.id', function ($query) {
            $query
                ->select(\DB::raw('MAX(id)'))
                ->from('test_session_service_runs')
                ->groupBy('service_id');
        });
    }

    public function activity(): HasMany
    {
        return $this->hasMany(SessionActivity::class);
    }

    public function reviewRequests(): HasMany
    {
        return $this->hasMany(ReviewRequest::class);
    }

    public function getManualTests()
    {
        return $this->services->flatMap(
            fn ($service) => $service->getManualTests()
        );
    }

    public function hasManualTests(): Attribute
    {
        return new Attribute(get: fn() => $this->getManualTests()->isNotEmpty());
    }
}
