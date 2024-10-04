<?php

namespace App\Models;

use App\Services\JUnitLogParser;
use Carbon\Carbon;
use Illuminate\Database\Eloquent\Casts\Attribute;
use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Carbon\CarbonInterval;

class SessionServiceRun extends Model
{
    protected $fillable = [
        'status',
        'passed',
        'failed',
        'skipped',
        'duration',
        'result_log',
        'run_log',
        'github_run_id',
        'commit_sha',
        'started_at',
        'finished_at',
    ];

    protected $table = 'test_session_service_runs';

    public function service(): BelongsTo
    {
        return $this->belongsTo(SessionService::class, 'service_id');
    }

    public function parsedResults(): Attribute
    {
        return new Attribute(get: fn() => new JUnitLogParser($this->result_log));
    }

    public function humanReadableTestDuration(): Attribute
    {
        return new Attribute(
            get: fn() => CarbonInterval::seconds($this->duration)
                ->cascade()
                ->forHumans(),
        );
    }

    public function humanReadableDuration(): Attribute
    {
        // calculate the duration from the started_at and finished_at timestamps
        return new Attribute(
            get: fn() => Carbon::make($this->started_at)
                ->diffAsCarbonInterval($this->finished_at)
                ->cascade()
                ->forHumans(),
        );
    }
}
