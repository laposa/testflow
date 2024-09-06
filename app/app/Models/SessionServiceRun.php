<?php

namespace App\Models;

use App\Services\JUnitLogParser;
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

    public function humanReadableDuration(): Attribute
    {
        return new Attribute(
            get: fn() => CarbonInterval::seconds($this->duration)
                ->cascade()
                ->forHumans(),
        );
    }
}
