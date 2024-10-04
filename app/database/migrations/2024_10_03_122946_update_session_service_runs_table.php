<?php

use Illuminate\Database\Migrations\Migration;
use Illuminate\Database\Schema\Blueprint;
use Illuminate\Support\Facades\Schema;

return new class extends Migration {
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('test_session_service_runs', function (Blueprint $table) {
            $table->bigInteger('github_run_id')->nullable();
            $table->string('commit_sha')->nullable();
            $table->dateTime('started_at')->nullable();
            $table->dateTime('finished_at')->nullable();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('test_session_service_runs', function (Blueprint $table) {
            $table->dropColumn(['github_run_id', 'commit_sha', 'started_at', 'finished_at']);
        });
    }
};
